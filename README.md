# Go MCP Intelligence Server

A Go MCP server using the official [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) SDK. It exposes bounded file reading, optional read-only PostgreSQL inspection, and analysis of operator-supplied network flow CSV data over stdio.

## Requirements

- Go 1.25 or newer. The selected MCP SDK release (v1.8.0) requires Go 1.25.0. Go's release policy supports each major release until two newer major releases are available.
- PostgreSQL is optional; database tools report a clear configuration error when `DATABASE_URL` is unset.

## Set up

```powershell
Copy-Item configs\.env.example .env
```

At startup the server loads `.env` from the current working directory when present. Existing process environment variables take precedence. Configure `MCP_ALLOWED_DIRS` with one or more trusted directories separated by the platform path-list separator. The defaults and limits are documented in [`configs/.env.example`](configs/.env.example).

Add private documents only under configured directories, and add an authorized flow export as `data/network/flows.csv`. The CSV must contain `source_ip,destination_ip,protocol,bytes,packets`. The checked-in example contains only documentation-range IP addresses and is not operational traffic.

For PostgreSQL, set `DATABASE_URL` to a dedicated account with read-only grants. The server verifies connection availability at startup when configured; it does not create schemas or modify data.

## Build and run

```powershell
go mod tidy
go test ./...
go build -o mcp-server.exe ./cmd/mcp-server
.\mcp-server.exe
```

Configure an MCP-compatible application to launch the executable as a stdio server. MCP protocol messages use stdin/stdout; operational logs are written to stderr. Press Ctrl+C for a graceful shutdown.

## Tools

| Tool | Purpose |
|---|---|
| `list_allowed_files` | List TXT and Markdown files below configured roots. |
| `read_file` | Read a UTF-8 TXT/Markdown file within a configured root. |
| `get_file_metadata` | Return path, byte size, supported format, and modification time. |
| `list_tables` | List base tables in the configured PostgreSQL schema. |
| `describe_table` | Inspect columns in the configured schema. |
| `query_readonly` | Execute one bounded SELECT/WITH query in a read-only transaction. |
| `get_traffic_summary` | Summarize authorized flow CSV data. |
| `get_top_talkers` | Rank source/destination IP addresses by bytes. |
| `get_protocol_statistics` | Aggregate flows, bytes, and packets by protocol. |

File extractors currently support only TXT and Markdown. PDF, DOCX, content search, and PCAP parsing are not implemented and are not represented as supported features.

## Configuration

| Variable | Default | Description |
|---|---:|---|
| `MCP_ALLOWED_DIRS` | required | Directories the file tools may access. |
| `MCP_MAX_FILE_BYTES` | `10485760` | Maximum file size read by file tools. |
| `MCP_MAX_LIST_FILES` | `200` | Maximum supported files returned by listing. |
| `DATABASE_URL` | unset | PostgreSQL DSN; leave empty to disable database tools. |
| `DATABASE_SCHEMA` | `public` | Schema exposed to inspection tools. |
| `DATABASE_QUERY_TIMEOUT` | `5s` | Database operation timeout. |
| `DATABASE_MAX_ROWS` | `500` | Maximum rows returned by a query. |
| `DATABASE_MAX_RESULT_BYTES` | `1048576` | Maximum returned text data per query. |
| `FLOW_CSV_PATH` | unset | Path to a flow CSV authorized for analysis. |
| `FLOW_MAX_FILE_BYTES` | `10485760` | Maximum size of the flow CSV. |

## Documentation

- [Architecture](docs/architecture.md)
- [Security](docs/security.md)
