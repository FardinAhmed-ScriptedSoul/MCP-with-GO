# Architecture

The process runs an MCP server over stdio using the official Go SDK. `cmd/mcp-server` loads configuration, constructs capability services, and routes lifecycle logs to stderr. `internal/server` registers MCP tool handlers; business logic remains in `internal/files`, `internal/database`, and `internal/network`.

The file service limits access to configured roots and extracts UTF-8 TXT and Markdown only. The PostgreSQL service is optional and wraps schema inspection and bounded queries in context deadlines and read-only transactions. The network service parses a configured structured flow CSV and calculates summary, top-talker, and protocol aggregates. It does not capture packets or parse PCAP files.

## Go and SDK versions

The module targets Go 1.25.0, the minimum required by official MCP Go SDK v1.8.0. Go's release policy supports a major release until two newer major releases are available; use a currently supported Go release for production builds.

## Data flow

```text
MCP client --stdio--> MCP SDK server --> tool handlers --> capability services
```

No tool uses file contents, database records, or flow data as executable instructions.
