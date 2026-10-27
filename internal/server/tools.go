package server

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/database"
	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/files"
	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/network"
)

type pathInput struct {
	Path string `json:"path" jsonschema:"File path returned by list_allowed_files"`
}

type readFileOutput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type fileListOutput struct {
	Files []files.Metadata `json:"files"`
}

type metadataOutput struct {
	Metadata files.Metadata `json:"metadata"`
}

type csvInput struct {
	Path     string `json:"path" jsonschema:"File path returned by list_allowed_files"`
	RowLimit int    `json:"row_limit,omitempty" jsonschema:"Maximum CSV data rows (1-500; default 100)"`
}

type csvOutput struct {
	Data files.CSVData `json:"data"`
}

type tableInput struct {
	Table string `json:"table" jsonschema:"Table name in the configured database schema"`
}

type queryInput struct {
	Query string `json:"query" jsonschema:"A single read-only SELECT or WITH query"`
}

type queryOutput struct {
	Result database.QueryResult `json:"result"`
}

type limitInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"Maximum talkers to return (1-50; default 10)"`
}

type summaryOutput struct {
	Summary network.Summary `json:"summary"`
}

type talkerOutput struct {
	Talkers []network.Talker `json:"talkers"`
}

type protocolOutput struct {
	Protocols []network.ProtocolStat `json:"protocols"`
}

func registerFileTools(s *mcp.Server, service *files.Service) {
	mcp.AddTool(s, &mcp.Tool{Name: "list_allowed_files", Description: "List supported text files under configured directories."},
		func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, fileListOutput, error) {
			result, err := service.ListAllowedFiles()
			return nil, fileListOutput{Files: result}, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "read_file", Description: "Read a bounded UTF-8 TXT or Markdown file within configured directories."},
		func(_ context.Context, _ *mcp.CallToolRequest, input pathInput) (*mcp.CallToolResult, readFileOutput, error) {
			content, err := service.ReadFile(input.Path)
			return nil, readFileOutput{Path: input.Path, Content: content}, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "get_file_metadata", Description: "Get metadata for a supported file within configured directories."},
		func(_ context.Context, _ *mcp.CallToolRequest, input pathInput) (*mcp.CallToolResult, metadataOutput, error) {
			meta, err := service.GetMetadata(input.Path)
			return nil, metadataOutput{Metadata: meta}, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "read_csv", Description: "Parse a bounded CSV file within configured directories and return its header and rows."},
		func(_ context.Context, _ *mcp.CallToolRequest, input csvInput) (*mcp.CallToolResult, csvOutput, error) {
			if input.RowLimit == 0 {
				input.RowLimit = 100
			}
			data, err := service.ReadCSV(input.Path, input.RowLimit)
			return nil, csvOutput{Data: data}, err
		})
}

func registerDatabaseTools(s *mcp.Server, service *database.Service) {
	unavailable := func() error {
		if service == nil {
			return fmt.Errorf("database tools are unavailable; configure DATABASE_URL")
		}
		return nil
	}
	mcp.AddTool(s, &mcp.Tool{Name: "list_tables", Description: "List base tables in the configured database schema."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string][]database.Table, error) {
			if err := unavailable(); err != nil {
				return nil, nil, err
			}
			tables, err := service.ListTables(ctx)
			return nil, map[string][]database.Table{"tables": tables}, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "describe_table", Description: "List columns for a table in the configured database schema."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input tableInput) (*mcp.CallToolResult, map[string][]database.Column, error) {
			if err := unavailable(); err != nil {
				return nil, nil, err
			}
			columns, err := service.DescribeTable(ctx, input.Table)
			return nil, map[string][]database.Column{"columns": columns}, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "query_readonly", Description: "Execute one bounded SELECT or WITH query using a read-only database transaction."},
		func(ctx context.Context, _ *mcp.CallToolRequest, input queryInput) (*mcp.CallToolResult, queryOutput, error) {
			if err := unavailable(); err != nil {
				return nil, queryOutput{}, err
			}
			result, err := service.QueryReadOnly(ctx, input.Query)
			return nil, queryOutput{Result: result}, err
		})
}

func registerNetworkTools(s *mcp.Server, service *network.Service) {
	mcp.AddTool(s, &mcp.Tool{Name: "get_traffic_summary", Description: "Summarize operator-supplied flow CSV data."},
		func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, summaryOutput, error) {
			result, err := service.Summary()
			return nil, summaryOutput{Summary: result}, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "get_top_talkers", Description: "Rank IP addresses in operator-supplied flow CSV data by bytes."},
		func(_ context.Context, _ *mcp.CallToolRequest, input limitInput) (*mcp.CallToolResult, talkerOutput, error) {
			if input.Limit == 0 {
				input.Limit = 10
			}
			result, err := service.TopTalkers(input.Limit)
			return nil, talkerOutput{Talkers: result}, err
		})
	mcp.AddTool(s, &mcp.Tool{Name: "get_protocol_statistics", Description: "Group flow counts, bytes, and packets by protocol."},
		func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, protocolOutput, error) {
			result, err := service.ProtocolStatistics()
			return nil, protocolOutput{Protocols: result}, err
		})
}
