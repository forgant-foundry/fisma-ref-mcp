# fisma-ref-mcp

Multi-corpus reference MCP server for AI-assisted compliance work. Provides semantic and deterministic access to three federal cybersecurity reference corpora — NIST SP 800-53 Rev 5, NIST CSF 2.0, and FY 2025 IG FISMA Metrics — enabling traceability from AI suggestions back to authoritative source material.

## Architecture

### Data model

| Layer | Technology | Purpose |
|---|---|---|
| Source | JSON files embedded in each corpus package | One package per document; replace JSON to update a corpus |
| Relational | SQLite in-memory (`modernc.org/sqlite`) | Deterministic lookups by ID, family, domain, function |
| Vector | chromem-go in-memory (`internal/vec_store/data/<model>/`) | Semantic search using pre-built embeddings across all corpora |

The relational DB is always populated at startup from the embedded JSON. The vector index is pre-built at developer build time and embedded in the binary — no embedding API calls happen at user startup. Without a pre-built index the binary falls back to FTS5 search automatically.

### Corpora

| Package | Source identifier | Contents |
|---|---|---|
| `internal/nist_800_53` | `nist_800_53` | NIST SP 800-53 Rev 5.2.0 — 1,196 controls across 20 families |
| `internal/nist_800_53b` | *(metadata only)* | NIST SP 800-53B 5.2.0 — baseline profiles (Low/Moderate/High/Privacy) |
| `internal/nist_csf` | `nist_csf_v2` | NIST CSF 2.0 — 185 subcategories across 6 functions |
| `internal/fisma` | `fisma_fy2025` | FY 2025 IG FISMA Metrics — 35 metrics with 5-level maturity model |

SP 800-53B is not a separate searchable corpus — it enriches SP 800-53 controls with baseline membership. Every `Control` returned by `get_control` and `get_baseline` includes a `baselines` field.

### Build-time embedding

Embeddings are generated once by the developer and committed to `internal/vec_store/data/<model>/`. The index covers all three corpora in a single chromem-go DB file.

```bash
make embed-nomic          # requires Ollama + nomic-embed-text:v1.5
make embed-qwen3          # requires Ollama + qwen3-embedding:4b
OPENAI_API_KEY=sk-... make embed-openai-small

# Then commit the updated index files
git add internal/vec_store/data/
git commit -m "update vector indexes"
```

The meta file (`chromem-meta.json`) records the provider and model. At startup the runtime validates that the runtime provider/model matches the index — a mismatch is a hard error.

### Execution modes

```
fisma-ref-mcp serve [--port 8080]   # HTTP MCP server (Streamable HTTP transport)
fisma-ref-mcp serve --stdio          # stdio MCP transport (Claude Desktop etc.)
fisma-ref-mcp search "<query>"       # cross-corpus semantic search → JSON stdout
fisma-ref-mcp search "<query>" --source nist_800_53|fisma_fy2025|nist_csf_v2
fisma-ref-mcp control <id>           # get NIST control by ID → JSON stdout
fisma-ref-mcp family [<id>]          # list families or controls in a family → JSON stdout
```

### MCP tools

| Tool | Description |
|---|---|
| `search` | Semantic (or FTS5 fallback) search across all corpora; `source` and `family` filters |
| `get_control` | Deterministic lookup by control ID (e.g. `AC-1`, `ac-2(1)`); includes baseline membership |
| `list_families` | All 20 NIST SP 800-53 control families |
| `get_family` | All base controls (no enhancements) in a family |
| `get_baseline` | All controls/enhancements in a SP 800-53B baseline (`low`, `moderate`, `high`, `privacy`) |
| `list_fisma_metrics` | FY 2025 IG FISMA metrics; optional domain filter |
| `get_fisma_metric` | Single metric by ID — full maturity levels, evidence, assessor notes, criteria refs |
| `get_metrics_by_control` | FISMA metrics that reference a given NIST SP 800-53 control ID |
| `list_csf_functions` | All 6 CSF 2.0 functions with their categories; optional function filter |
| `get_csf_subcategory` | Single CSF 2.0 subcategory by ID with implementation examples |

### Embedding configuration

The embedding provider and model are auto-detected from the `chromem-meta.json` embedded in the binary. Users never pass provider or model flags. The only runtime inputs are:

| Env var | When required |
|---|---|
| `OPENAI_API_KEY` | `embed_openai_small` binary variant |
| `OLLAMA_URL` | `embed_nomic` or `embed_qwen3` variant; default `http://localhost:11434` |

## Package layout

```
cmd/
  root.go         buildStore helper; no persistent flags beyond help
  serve.go        HTTP and stdio MCP server
  search.go       search subcommand (--source, --limit)
  control.go      control subcommand
  family.go       family subcommand

internal/
  nist_800_53/    NIST SP 800-53 types, OSCAL JSON parsing, Load(), NormalizeID()
    data/         nist-800-53r5.json
  nist_800_53b/   SP 800-53B baseline profiles, Load() → map[controlID][]baseline
    data/         nist-800-53b.json
  nist_csf/       NIST CSF 2.0 types, flat-graph JSON parsing, Load()
    data/         nist-csf-2.0.json
  fisma/          FY 2025 IG FISMA types, JSON parsing, Load(), ContextMarkdown
    data/         fy2025-ig-fisma-metrics.json, fy2025-ig-fisma-metrics-context.md
  vec_store/      VectorMeta, PrebuiltVector(), build-tag embed files
    data/
      nomic/      chromem.db + chromem-meta.json (nomic-embed-text:v1.5)
      qwen3/      chromem.db + chromem-meta.json (qwen3-embedding:4b)
      openai-small/ chromem.db + chromem-meta.json (text-embedding-3-small)
  rel_store/      Unified data access layer
    store.go      Store struct, Config, public API
    relational.go in-memory SQLite: schema, seed, FTS5 queries
    vector.go     chromem-go: multi-collection index, semantic search
    vector_stub.go stub for builds without vector support
  mcp/
    server.go     NewServer, ServeHTTP, ServeStdio, all tool handlers

tools/
  gen-embeddings/main.go    //go:build ignore; indexes all three corpora
  parse-fisma-metrics/      Python PDF parser for the IG FISMA metrics document
```

## Go conventions

### Module

`github.com/forgant-foundry/fisma-ref-mcp`

### Dependencies

- `github.com/mark3labs/mcp-go` — MCP protocol (stdio + Streamable HTTP transports)
- `github.com/philippgille/chromem-go` — in-process vector DB
- `modernc.org/sqlite` — **pure-Go SQLite; no CGO, no system lib dependency**
- `github.com/spf13/cobra` — CLI

### Rules

- **No CGO.** Use `modernc.org/sqlite`, not `mattn/go-sqlite3`.
- Errors always wrapped: `fmt.Errorf("context doing X: %w", err)`.
- No global state. Dependencies injected via constructors (`rel_store.New`, `mcp.NewServer`).
- `context.Context` propagated to every DB call and HTTP handler.
- Table-driven tests with `t.Run`; integration tests use an in-memory store (no mocking the DB).
- `//go:embed` directives on `var` declarations at package level, not inside functions.
- Return concrete types from constructors; use interfaces only at function parameter boundaries.
- `CallToolRequest` helper methods: `req.RequireString`, `req.GetString`, `req.GetInt` — do not index `req.Params.Arguments` directly (it is `any`).
- Build tags for embedding variants: `embed_nomic`, `embed_qwen3`, `embed_openai_small`. Untagged builds fall back to FTS5 (no vector search). The `no_embeddings` tag is no longer used.
- Source identifiers are versioned strings: `nist_800_53`, `fisma_fy2025`, `nist_csf_v2`. Follow this pattern when adding new corpora.

### Adding a new corpus

1. Create `internal/<source_id>/` with `types.go`, `embed.go` (with `//go:embed data/`), and `data/<source>.json`
2. Add FTS5 table and seed function to `internal/rel_store/relational.go`
3. Add a new collection constant and collection handling to `internal/rel_store/vector.go` and `vector_stub.go`
4. Add the new source to `internal/rel_store/store.go` (`New()` and public query methods)
5. Register MCP tools in `internal/mcp/server.go`
6. Update `tools/gen-embeddings/main.go` to build the new collection
7. Run `make embed-<model>` to regenerate and commit the updated index

### Forglet

`go.mod` is managed by forglet. After any `forglet synth`:
```
chmod 644 go.mod
go get <any new deps>
go mod tidy
```
