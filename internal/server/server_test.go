package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/files"
	"github.com/FardinAhmed-ScriptedSoul/MCP-with-GO/internal/network"
)

func TestServerDiscoversAndInvokesTools(t *testing.T) {
	root := t.TempDir()
	document := filepath.Join(root, "welcome.md")
	if err := os.WriteFile(document, []byte("# welcome"), 0600); err != nil {
		t.Fatal(err)
	}
	csvDocument := filepath.Join(root, "records.csv")
	if err := os.WriteFile(csvDocument, []byte("name,value\nAda,1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	flowPath := filepath.Join(root, "flows.csv")
	flowData := "source_ip,destination_ip,protocol,bytes,packets\n192.0.2.1,198.51.100.1,TCP,42,3\n"
	if err := os.WriteFile(flowPath, []byte(flowData), 0600); err != nil {
		t.Fatal(err)
	}
	fileService, err := files.NewService([]string{root}, 1024, 20)
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mcpServer := New(fileService, nil, network.NewService(flowPath, 1024), logger)

	ctx, cancel := context.WithCancel(context.Background())
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverDone := make(chan error, 1)
	go func() { serverDone <- mcpServer.Run(ctx, serverTransport) }()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		cancel()
		t.Fatalf("connect MCP client: %v", err)
	}
	defer func() {
		_ = session.Close()
		cancel()
		<-serverDone
	}()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools.Tools) != 10 {
		t.Fatalf("ListTools() returned %d tools, want 10", len(tools.Tools))
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "read_csv",
		Arguments: map[string]string{"path": csvDocument},
	})
	if err != nil {
		t.Fatalf("call read_csv: %v", err)
	}
	if result.IsError {
		t.Fatalf("read_csv returned tool error: %v", result.Content[0].(*mcp.TextContent).Text)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_traffic_summary",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("call get_traffic_summary: %v", err)
	}
	if result.IsError {
		t.Fatalf("get_traffic_summary returned tool error: %+v", result.Content)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_tables",
		Arguments: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("call list_tables: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected unavailable database tool to return a tool error")
	}
}
