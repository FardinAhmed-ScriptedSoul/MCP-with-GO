package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvAndRespectProcessEnvironment(t *testing.T) {
	root := t.TempDir()
	envFile := filepath.Join(root, ".env")
	if err := os.WriteFile(envFile, []byte("MCP_MAX_FILE_BYTES=123\nDATABASE_SCHEMA=from-dotenv\n"), 0600); err != nil {
		t.Fatal(err)
	}
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	setOrUnset(t, "MCP_MAX_FILE_BYTES", "")
	t.Setenv("MCP_ALLOWED_DIRS", root)
	t.Setenv("DATABASE_SCHEMA", "from-process")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("FLOW_CSV_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxFileBytes != 123 {
		t.Errorf("MaxFileBytes = %d, want dotenv value 123", cfg.MaxFileBytes)
	}
	if cfg.DatabaseSchema != "from-process" {
		t.Errorf("DatabaseSchema = %q, want process environment value", cfg.DatabaseSchema)
	}
}

func setOrUnset(t *testing.T, name, value string) {
	t.Helper()
	old, existed := os.LookupEnv(name)
	if err := os.Unsetenv(name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			if err := os.Setenv(name, old); err != nil {
				t.Errorf("restore %s: %v", name, err)
			}
		} else if err := os.Unsetenv(name); err != nil {
			t.Errorf("clear %s: %v", name, err)
		}
	})
	if value != "" {
		t.Setenv(name, value)
	}
}
