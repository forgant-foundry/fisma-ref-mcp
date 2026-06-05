# fisma-ref-mcp

NIST SP 800-53 Rev 5 reference MCP server for AI-assisted compliance work. Provides semantic and deterministic access to official control text, enabling traceability from AI suggestions back to authoritative NIST source material.

## Architecture

### Data model

| Layer | Technology | Purpose |
|---|---|---|
| Source | `internal/nist/data/nist-800-53r5.json` (embedded) | Single source of truth; replace with official NIST OSCAL export |
| Relational | SQLite in-memory (`modernc.org/sqlite`) | Deterministic lookups by control ID, family, revision |
| Vector | chromem-go in-memory (`internal/nist/data/chromem.db` embedded) | Semantic search using pre-built embeddings |

The relational DB is always populated at startup from the embedded JSON. The vector index is pre-built at developer build time and embedded in the binary — no embedding API calls happen at user startup.

### Build-time embedding

Embeddings are generated once by the developer and committed as `internal/nist/data/chromem.db`:

```bash
# Generate with OpenAI (recommended for best quality)
OPENAI_API_KEY=sk-... go run ./tools/gen-embeddings/main.go --provider openai

# Generate with local Ollama
go run ./tools/gen-embeddings/main.go --provider ollama --model nomic-embed-text

# Or via go generate
EMBEDDING_PROVIDER=openai OPENAI_API_KEY=sk-... go generate ./internal/nist

# Then rebuild the binary to embed the new index
go build .
```

The meta file (`internal/nist/data/chromem-meta.json`) records which provider and model were used. At startup the runtime validates that the configured provider/model matches; a mismatch is a hard error because vectors in different embedding spaces produce garbage results.

Without a pre-built index the binary falls back to SQL `LIKE` search automatically.

### Execution modes

```
fisma-ref-mcp serve [--port 8080]   # HTTP MCP server (JSON-RPC 2.0 + SSE)
fisma-ref-mcp serve --stdio          # stdio MCP transport (Claude Desktop etc.)
fisma-ref-mcp search "<query>"       # single-shot semantic search → JSON stdout
fisma-ref-mcp control <id>           # get control by ID → JSON stdout
fisma-ref-mcp family [<id>]          # list families or controls in a family → JSON stdout
```

### MCP tools

| Tool | Description |
|---|---|
| `search_controls` | Semantic (or SQL fallback) search with optional family filter |
| `get_control` | Deterministic lookup by control ID (e.g. `AC-1`, `ac-2(1)`) |
| `list_families` | All 20 control families |
| `get_family` | All base controls (no enhancements) in a family |

### Embedding configuration

Vector search requires an embedding provider. Without one the server falls back to SQL `LIKE` search transparently.

| Flag | Env var | Values |
|---|---|---|
| `--embedding-provider` | `EMBEDDING_PROVIDER` | `openai`, `ollama` |
| `--embedding-model` | `EMBEDDING_MODEL` | provider-specific; defaults to `text-embedding-3-small` / `nomic-embed-text` |
| `--ollama-url` | `OLLAMA_URL` | default `http://localhost:11434` |
| *(none)* | `OPENAI_API_KEY` | required when provider is `openai` |

## Package layout

```
cmd/            cobra commands; no business logic
  root.go       persistent flags, buildStore helper
  serve.go      HTTP and stdio MCP server
  search.go     search subcommand
  control.go    control subcommand
  family.go     family subcommand

internal/
  nist/         NIST data types, OSCAL JSON parsing, embed
    types.go    Control, Family, Part, Catalog — OSCAL-compatible
    embed.go    //go:embed + Load() + PrebuiltVector()
    data/       nist-800-53r5.json, chromem.db, chromem-meta.json

tools/
  gen-embeddings/main.go   //go:build ignore; run via go generate or go run

  store/        unified data access layer
    store.go    Store struct, Config, public API
    relational.go  in-memory SQLite: schema, seed, queries
    vector.go   chromem-go: index build, semantic search

  mcp/
    server.go   NewServer, ServeHTTP, ServeStdio, tool handlers
```

## Go conventions

### Module

`github.com/forgant-foundry/fisma-ref-mcp`

### Dependencies

- `github.com/mark3labs/mcp-go` — MCP protocol (stdio + HTTP/SSE transports)
- `github.com/philippgille/chromem-go` — in-process vector DB
- `modernc.org/sqlite` — **pure-Go SQLite; no CGO, no system lib dependency**
- `github.com/spf13/cobra` — CLI

### Rules

- **No CGO.** Use `modernc.org/sqlite`, not `mattn/go-sqlite3`. This keeps cross-compilation simple and the binary self-contained.
- Errors always wrapped: `fmt.Errorf("context doing X: %w", err)`.
- No global state. Dependencies injected via constructors (`store.New`, `mcp.NewServer`).
- `context.Context` propagated to every DB call and HTTP handler.
- Table-driven tests with `t.Run`; integration tests use an in-memory store (no mocking the DB).
- `//go:embed` directives on `var` declarations at package level, not inside functions.
- Return concrete types from constructors; use interfaces only at function parameter boundaries.
- `CallToolRequest` helper methods: `req.RequireString`, `req.GetString`, `req.GetInt` — do not index `req.Params.Arguments` directly (it is `any`).

### Forglet

`go.mod` is managed by forglet. After any `forglet synth`:
```
chmod 644 go.mod
go get <any new deps>
go mod tidy
```

### Adding the real NIST data

Replace `internal/nist/data/nist-800-53r5.json` with the official NIST OSCAL export:
- Source: https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final (OSCAL JSON)
- The embedded placeholder contains only AC and SI families for development.
- The `nist.Catalog` types follow OSCAL 1.x schema; verify field names if switching formats.
