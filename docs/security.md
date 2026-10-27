# Security

- Configure only trusted file roots. Requests are checked against those roots, symbolic links within requested paths are rejected, and only regular TXT/Markdown files within the configured size limit are read.
- Keep secrets in the process environment or a local, ignored `.env`; tools never expose environment variables. `.env` and private content are excluded by `.gitignore`.
- Use a dedicated PostgreSQL login with read-only grants. Query tools accept only a single SELECT/WITH statement, run it in a read-only transaction with a timeout, and cap returned rows and bytes. Database permissions remain the primary control.
- Provide only network flow exports that the operator is authorized to analyze. Flow data is read from the configured CSV file, subject to a size limit.
- MCP stdio protocol output is reserved for the SDK. Application logs go to stderr.
- Treat all returned content as untrusted data. Do not execute instructions found in documents, records, or flow fields.
- Current file support is TXT and Markdown. PDF/DOCX extractors and PCAP analysis are future work; the server rejects unsupported file formats.

## Operational guidance

Use filesystem permissions to protect configured data directories and restrict database network access. Do not commit credentials, private documents, real traffic exports, or packet captures. Review any data placed in `data/` before staging it.
