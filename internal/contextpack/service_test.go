package contextpack

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/database"
	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/files"
)

type schemaStub struct {
	tables    []database.TableSchema
	truncated bool
	err       error
}

func (s schemaStub) SchemaSnapshot(context.Context, int) ([]database.TableSchema, bool, error) {
	return s.tables, s.truncated, s.err
}

func TestBuildRanksAndCitesFileCSVAndSchemaEvidence(t *testing.T) {
	root := t.TempDir()
	markdown := filepath.Join(root, "guide.md")
	if err := os.WriteFile(markdown, []byte("# Importing\nThe customer import fails when email is missing.\nUnrelated notes about gardening.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	csvPath := filepath.Join(root, "customers.csv")
	if err := os.WriteFile(csvPath, []byte("customer_id,email,status\n7,ada@example.test,invalid\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fileService, err := files.NewService([]string{root}, 4096, 20)
	if err != nil {
		t.Fatal(err)
	}
	databaseStub := schemaStub{tables: []database.TableSchema{{
		Schema: "public",
		Name:   "customers",
		Columns: []database.Column{
			{Name: "email", Type: "text"},
			{Name: "status", Type: "text", Nullable: true},
		},
	}}}
	service, err := NewService(fileService, databaseStub, 1000, 4096)
	if err != nil {
		t.Fatal(err)
	}

	pack, err := service.Build(context.Background(), "customer email import status", 500, 10)
	if err != nil {
		t.Fatal(err)
	}
	if pack.EstimatedTokens > pack.TokenBudget {
		t.Fatalf("estimated tokens %d exceed budget %d", pack.EstimatedTokens, pack.TokenBudget)
	}
	if pack.FilesScanned != 2 || len(pack.Items) < 3 {
		t.Fatalf("expected file, CSV, and schema evidence; got %+v", pack)
	}
	if !strings.Contains(pack.Notice, "untrusted") {
		t.Fatalf("missing untrusted evidence notice: %q", pack.Notice)
	}
	sources := make(map[string]bool)
	for _, item := range pack.Items {
		sources[item.Source] = true
	}
	if !sources[markdown+":line 2"] || !sources[csvPath+":row 2"] || !sources["database:public.customers"] {
		t.Fatalf("missing expected citations: %+v", sources)
	}
}

func TestBuildHonorsScanAndResultBudgets(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "large.txt")
	if err := os.WriteFile(path, []byte("database database database database database database database database\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fileService, err := files.NewService([]string{root}, 4096, 20)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(fileService, nil, 100, 8)
	if err != nil {
		t.Fatal(err)
	}
	pack, err := service.Build(context.Background(), "database", 100, 1)
	if err != nil {
		t.Fatal(err)
	}
	if pack.FilesScanned != 0 || !pack.Truncated || len(pack.Items) != 0 {
		t.Fatalf("expected scan cap to skip the file: %+v", pack)
	}

	longRoot := t.TempDir()
	longPath := filepath.Join(longRoot, "notes.txt")
	longLine := "database " + strings.Repeat("relevant ", 200)
	if err := os.WriteFile(longPath, []byte(longLine), 0600); err != nil {
		t.Fatal(err)
	}
	longFiles, err := files.NewService([]string{longRoot}, 4096, 20)
	if err != nil {
		t.Fatal(err)
	}
	budgetService, err := NewService(longFiles, nil, 100, 4096)
	if err != nil {
		t.Fatal(err)
	}
	bounded, err := budgetService.Build(context.Background(), "database", 100, 1)
	if err != nil {
		t.Fatal(err)
	}
	if bounded.EstimatedTokens > 100 || !bounded.Truncated || len(bounded.Items) != 1 {
		t.Fatalf("expected excerpt to be truncated to budget: %+v", bounded)
	}
}

func TestBuildRejectsInvalidLimitsAndReportsSourceErrors(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "bad.csv")
	if err := os.WriteFile(path, []byte("name,value\nAda\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fileService, err := files.NewService([]string{root}, 4096, 20)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(fileService, nil, 200, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Build(context.Background(), "!!!", 0, 0); err == nil {
		t.Fatal("expected query with no searchable words to be rejected")
	}
	if _, err := service.Build(context.Background(), "name", 50, 0); err == nil {
		t.Fatal("expected token budget validation error")
	}
	if _, err := service.Build(context.Background(), "Ada", 0, 0); err == nil {
		t.Fatal("expected malformed CSV error")
	}
}
