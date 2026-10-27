package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AllowedDirs              []string
	MaxFileBytes             int64
	MaxListFiles             int
	DatabaseURL              string
	DatabaseSchema           string
	DatabaseQueryTimeout     time.Duration
	DatabaseMaxRows          int
	DatabaseMaxResultBytes   int
	FlowCSVPath              string
	FlowMaxFileBytes         int64
	ContextPackDefaultTokens int
	ContextPackMaxScanBytes  int64
}

func Load() (Config, error) {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			return Config{}, fmt.Errorf("load .env: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("inspect .env: %w", err)
	}

	var cfg Config
	dirs := filepath.SplitList(os.Getenv("MCP_ALLOWED_DIRS"))
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		abs, err := filepath.Abs(strings.TrimSpace(dir))
		if err != nil {
			return Config{}, fmt.Errorf("resolve allowed directory: %w", err)
		}
		cfg.AllowedDirs = append(cfg.AllowedDirs, filepath.Clean(abs))
	}
	if len(cfg.AllowedDirs) == 0 {
		return Config{}, fmt.Errorf("MCP_ALLOWED_DIRS must contain at least one directory")
	}

	var err error
	if cfg.MaxFileBytes, err = int64Env("MCP_MAX_FILE_BYTES", 10<<20); err != nil {
		return Config{}, err
	}
	if cfg.MaxListFiles, err = intEnv("MCP_MAX_LIST_FILES", 200); err != nil {
		return Config{}, err
	}
	if cfg.FlowMaxFileBytes, err = int64Env("FLOW_MAX_FILE_BYTES", 10<<20); err != nil {
		return Config{}, err
	}
	if cfg.ContextPackDefaultTokens, err = intEnv("CONTEXT_PACK_DEFAULT_TOKENS", 2000); err != nil {
		return Config{}, err
	}
	if cfg.ContextPackMaxScanBytes, err = int64Env("CONTEXT_PACK_MAX_SCAN_BYTES", 10<<20); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseQueryTimeout, err = durationEnv("DATABASE_QUERY_TIMEOUT", 5*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMaxRows, err = intEnv("DATABASE_MAX_ROWS", 500); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMaxResultBytes, err = intEnv("DATABASE_MAX_RESULT_BYTES", 1<<20); err != nil {
		return Config{}, err
	}
	for name, value := range map[string]int64{
		"MCP_MAX_FILE_BYTES":          cfg.MaxFileBytes,
		"FLOW_MAX_FILE_BYTES":         cfg.FlowMaxFileBytes,
		"CONTEXT_PACK_MAX_SCAN_BYTES": cfg.ContextPackMaxScanBytes,
	} {
		if value <= 0 {
			return Config{}, fmt.Errorf("%s must be greater than zero", name)
		}
	}
	if cfg.MaxListFiles <= 0 || cfg.DatabaseMaxRows <= 0 || cfg.DatabaseMaxResultBytes <= 0 ||
		cfg.DatabaseQueryTimeout <= 0 || cfg.ContextPackDefaultTokens < 100 || cfg.ContextPackDefaultTokens > 8000 {
		return Config{}, fmt.Errorf("file, database, and context-pack limits must be positive; CONTEXT_PACK_DEFAULT_TOKENS must be between 100 and 8000")
	}

	cfg.DatabaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	cfg.DatabaseSchema = strings.TrimSpace(os.Getenv("DATABASE_SCHEMA"))
	if cfg.DatabaseSchema == "" {
		cfg.DatabaseSchema = "public"
	}
	flowPath := strings.TrimSpace(os.Getenv("FLOW_CSV_PATH"))
	if flowPath != "" {
		cfg.FlowCSVPath, err = filepath.Abs(flowPath)
		if err != nil {
			return Config{}, fmt.Errorf("resolve FLOW_CSV_PATH: %w", err)
		}
	}
	return cfg, nil
}

func intEnv(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return n, nil
}

func int64Env(name string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return n, nil
}

func durationEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", name, err)
	}
	return d, nil
}
