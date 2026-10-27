package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Service struct {
	db             *sql.DB
	schema         string
	queryTimeout   time.Duration
	maxRows        int
	maxResultBytes int
}

type Table struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

type TableSchema struct {
	Schema  string   `json:"schema"`
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

type QueryResult struct {
	Columns   []string `json:"columns"`
	Rows      [][]any  `json:"rows"`
	Truncated bool     `json:"truncated"`
}

func NewService(ctx context.Context, dsn, schema string, timeout time.Duration, maxRows, maxResultBytes int) (*Service, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, nil
	}
	if schema == "" || timeout <= 0 || maxRows <= 0 || maxResultBytes <= 0 {
		return nil, fmt.Errorf("database schema and positive query limits are required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL connection: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	return &Service{db: db, schema: schema, queryTimeout: timeout, maxRows: maxRows, maxResultBytes: maxResultBytes}, nil
}

func (s *Service) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Service) ListTables(ctx context.Context) ([]Table, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	rows, err := s.db.QueryContext(ctx,
		`SELECT table_schema, table_name FROM information_schema.tables
		 WHERE table_schema = $1 AND table_type = 'BASE TABLE' ORDER BY table_name`, s.schema)
	if err != nil {
		return nil, fmt.Errorf("list database tables: %w", err)
	}
	defer rows.Close()
	var tables []Table
	for rows.Next() {
		var table Table
		if err := rows.Scan(&table.Schema, &table.Name); err != nil {
			return nil, fmt.Errorf("read table metadata: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate table metadata: %w", err)
	}
	return tables, nil
}

func (s *Service) DescribeTable(ctx context.Context, table string) ([]Column, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	rows, err := s.db.QueryContext(ctx,
		`SELECT column_name, data_type, is_nullable FROM information_schema.columns
		 WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`, s.schema, table)
	if err != nil {
		return nil, fmt.Errorf("describe table: %w", err)
	}
	defer rows.Close()
	var columns []Column
	for rows.Next() {
		var column Column
		var nullable string
		if err := rows.Scan(&column.Name, &column.Type, &nullable); err != nil {
			return nil, fmt.Errorf("read column metadata: %w", err)
		}
		column.Nullable = nullable == "YES"
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate column metadata: %w", err)
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("table %q was not found in schema %q", table, s.schema)
	}
	return columns, nil
}

func (s *Service) SchemaSnapshot(ctx context.Context, maxColumns int) ([]TableSchema, bool, error) {
	if maxColumns < 1 || maxColumns > 5000 {
		return nil, false, fmt.Errorf("schema column limit must be between 1 and 5000")
	}
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	rows, err := s.db.QueryContext(ctx,
		`SELECT table_name, column_name, data_type, is_nullable
		 FROM information_schema.columns
		 WHERE table_schema = $1
		 ORDER BY table_name, ordinal_position
		 LIMIT $2`, s.schema, maxColumns+1)
	if err != nil {
		return nil, false, fmt.Errorf("read database schema snapshot: %w", err)
	}
	defer rows.Close()
	snapshot := make([]TableSchema, 0)
	tableIndex := make(map[string]int)
	columnCount := 0
	truncated := false
	for rows.Next() {
		if columnCount == maxColumns {
			truncated = true
			break
		}
		var tableName, columnName, dataType, nullable string
		if err := rows.Scan(&tableName, &columnName, &dataType, &nullable); err != nil {
			return nil, false, fmt.Errorf("read schema snapshot row: %w", err)
		}
		index, ok := tableIndex[tableName]
		if !ok {
			index = len(snapshot)
			tableIndex[tableName] = index
			snapshot = append(snapshot, TableSchema{Schema: s.schema, Name: tableName})
		}
		snapshot[index].Columns = append(snapshot[index].Columns, Column{
			Name:     columnName,
			Type:     dataType,
			Nullable: nullable == "YES",
		})
		columnCount++
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate schema snapshot: %w", err)
	}
	return snapshot, truncated, nil
}

func (s *Service) QueryReadOnly(ctx context.Context, query string) (QueryResult, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" || !isReadQuery(trimmed) {
		return QueryResult{}, fmt.Errorf("only a single SELECT or WITH query is allowed")
	}
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return QueryResult{}, fmt.Errorf("begin read-only transaction: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, trimmed)
	if err != nil {
		return QueryResult{}, fmt.Errorf("execute read-only query: %w", err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return QueryResult{}, fmt.Errorf("read query columns: %w", err)
	}
	result := QueryResult{Columns: columns, Rows: make([][]any, 0)}
	resultBytes := 0
	for rows.Next() {
		if len(result.Rows) == s.maxRows {
			result.Truncated = true
			break
		}
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return QueryResult{}, fmt.Errorf("read query row: %w", err)
		}
		for i, value := range values {
			if bytes, ok := value.([]byte); ok {
				values[i] = string(bytes)
			}
		}
		encodedSize := 0
		for _, value := range values {
			encodedSize += len(fmt.Sprint(value))
		}
		if resultBytes+encodedSize > s.maxResultBytes {
			result.Truncated = true
			break
		}
		resultBytes += encodedSize
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return QueryResult{}, fmt.Errorf("iterate query rows: %w", err)
	}
	return result, nil
}

func isReadQuery(query string) bool {
	upper := strings.ToUpper(query)
	if strings.Contains(upper, ";") || strings.Contains(upper, "--") || strings.Contains(upper, "/*") || strings.Contains(upper, "*/") {
		return false
	}
	first := strings.Fields(upper)
	if len(first) == 0 || (first[0] != "SELECT" && first[0] != "WITH") {
		return false
	}
	return true
}
