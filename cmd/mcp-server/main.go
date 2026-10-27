package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/config"
	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/contextpack"
	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/database"
	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/files"
	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/network"
	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fileService, err := files.NewService(cfg.AllowedDirs, cfg.MaxFileBytes, cfg.MaxListFiles)
	if err != nil {
		return err
	}

	dbService, err := database.NewService(
		context.Background(),
		cfg.DatabaseURL,
		cfg.DatabaseSchema,
		cfg.DatabaseQueryTimeout,
		cfg.DatabaseMaxRows,
		cfg.DatabaseMaxResultBytes,
	)
	if err != nil {
		return err
	}
	if dbService != nil {
		defer dbService.Close()
	}

	networkService := network.NewService(cfg.FlowCSVPath, cfg.FlowMaxFileBytes)
	contextPackService, err := contextpack.NewService(fileService, dbService, cfg.ContextPackDefaultTokens, cfg.ContextPackMaxScanBytes)
	if err != nil {
		return err
	}
	mcpServer := server.New(fileService, dbService, networkService, contextPackService, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, mcpServer); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
