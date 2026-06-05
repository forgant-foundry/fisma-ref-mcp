# fisma-ref-mcp

A local [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server that gives AI assistants direct, citeable access to **NIST SP 800-53 Rev 5** security and privacy controls.

Install it once, point your AI at it, and every suggestion about access control, audit logging, encryption, or identity comes with a traceable reference to the exact control that requires it — not a paraphrase, the official text.

---

## Why this exists

AI assistants are useful for compliance work — drafting SSP narratives, reviewing architecture decisions, writing policy, generating audit checklists. The problem is that they paraphrase controls from training data, which gets the spirit right and the specifics wrong. Wrong control IDs, wrong parameters, wrong baseline applicability. An assessor reviewing your SSP will find every one of those mistakes.

This tool eliminates that problem by embedding the full control catalog directly in the binary and exposing it to AI tools via MCP. The AI queries the actual source; every citation is exact.

---

## NIST SP 800-53 Data

| Field | Value |
|---|---|
| Document | Security and Privacy Controls for Information Systems and Organizations |
| Identifier | SP 800-53 Rev 5 |
| Version | **5.2.0** |
| Source | [csrc.nist.gov](https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final) |
| Base controls | 324 |
| Control enhancements | 872 |
| **Total** | **1,196** |
| Families | 20 |

### Control families

| ID | Family |
|---|---|
| AC | Access Control |
| AT | Awareness and Training |
| AU | Audit and Accountability |
| CA | Assessment, Authorization, and Monitoring |
| CM | Configuration Management |
| CP | Contingency Planning |
| IA | Identification and Authentication |
| IR | Incident Response |
| MA | Maintenance |
| MP | Media Protection |
| PE | Physical and Environmental Protection |
| PL | Planning |
| PM | Program Management |
| PS | Personnel Security |
| PT | Personally Identifiable Information Processing and Transparency |
| RA | Risk Assessment |
| SA | System and Services Acquisition |
| SC | System and Communications Protection |
| SI | System and Information Integrity |
| SR | Supply Chain Risk Management |

> **Note:** This dataset reflects **SP 800-53 5.2.0**, which superseded the original Rev 5 release. It does not include NIST SP 800-53B baseline designations (Low/Moderate/High impact level assignments) or FedRAMP-specific parameter overlays. See [Limitations](#limitations) for details.

---

## Installation

### Pre-built binaries

Download the latest release for your platform from the [releases page](https://github.com/forgant-foundry/fisma-ref-mcp/releases). Each release ships four binary variants — pick the one that matches how you want to run it:

| Variant | Suffix | Search | Requires at runtime |
|---|---|---|---|
| **nomic** | `_nomic` | Vector + FTS5 | Ollama running locally |
| **qwen3** | `_qwen3` | Vector + FTS5 | Ollama running locally |
| **openai-small** | `_openai-small` | Vector + FTS5 | `OPENAI_API_KEY` |
| **slim** | `_slim` | FTS5 only | Nothing |

**Vector search** uses pre-built semantic embeddings to find controls by meaning, even when the exact words don't appear in the control text. **FTS5** is full-text keyword search with BM25 relevance ranking — fast, offline, and no configuration required. Both are useful; vector search adds value for open-ended conceptual queries.

If you don't have Ollama and don't want to manage an OpenAI key, the slim build is fully self-contained and works well for most lookups.

### Build from source

Requires Go 1.25+.

```bash
git clone https://github.com/forgant-foundry/fisma-ref-mcp
cd fisma-ref-mcp
go build -o fisma-ref-mcp .           # default (uses committed vector index)
go build -tags embed_nomic    -o fisma-ref-mcp .
go build -tags embed_qwen3    -o fisma-ref-mcp .
go build -tags embed_openai_small -o fisma-ref-mcp .
go build -tags no_embeddings  -o fisma-ref-mcp .   # slim, FTS5 only
```

---

## MCP server setup

### Claude Desktop (stdio transport)

Add to your `claude_desktop_config.json`. The binary automatically detects its embedded vector index — no embedding flags required.

**nomic or qwen3 variant** (Ollama must be running):

```json
{
  "mcpServers": {
    "fisma-ref": {
      "command": "/path/to/fisma-ref-mcp",
      "args": ["serve", "--stdio"]
    }
  }
}
```

**openai-small variant**:

```json
{
  "mcpServers": {
    "fisma-ref": {
      "command": "/path/to/fisma-ref-mcp",
      "args": ["serve", "--stdio"],
      "env": {
        "OPENAI_API_KEY": "sk-..."
      }
    }
  }
}
```

**slim variant** (no configuration needed at all):

```json
{
  "mcpServers": {
    "fisma-ref": {
      "command": "/path/to/fisma-ref-mcp",
      "args": ["serve", "--stdio"]
    }
  }
}
```

### HTTP server

For tools that connect over HTTP (VS Code extensions, custom agents, etc.):

```bash
fisma-ref-mcp serve --port 8080
```

The server exposes a [Streamable HTTP MCP](https://modelcontextprotocol.io/docs/concepts/transports) endpoint at `http://localhost:8080`.

---

## MCP tools

Once connected, the AI has access to four tools.

### `search_controls`

Searches across all control text (title, statement, and supplemental guidance). Uses semantic vector search when the binary includes a pre-built index; falls back to FTS5 full-text search otherwise. Both modes return results ranked by relevance.

```
query   string  (required) Natural-language description of what you are looking for
limit   number  (optional) Max results, default 10, max 50
family  string  (optional) Filter to a specific family, e.g. "AC"
```

**Example prompts:**
- *"What controls cover multi-factor authentication?"*
- *"Show me controls related to audit log retention in the AU family"*
- *"What does NIST require for supply chain risk?"*

### `get_control`

Retrieve the full text of a single control by ID.

```
id  string  (required) Control ID, e.g. "AC-1", "SI-3", "AC-2(1)"
```

**Example prompts:**
- *"Get me the full text of AC-17"*
- *"What does SI-7(1) say?"*

### `list_families`

Returns all 20 control families with their IDs and titles. Useful for exploration and orientation.

### `get_family`

Returns all base controls (no enhancements) in a given family.

```
id  string  (required) Two-letter family ID, e.g. "AC", "SI"
```

**Example prompts:**
- *"What controls are in the IA family?"*
- *"List all audit controls"*

---

## CLI usage

Every MCP tool is also available as a standalone command that prints JSON to stdout — useful for scripting, piping into `jq`, or quick lookups without starting a server.

```bash
# List all control families
fisma-ref-mcp family

# List all controls in a family
fisma-ref-mcp family AC

# Get a single control (case-insensitive, padded or unpadded)
fisma-ref-mcp control AC-1
fisma-ref-mcp control ac-17
fisma-ref-mcp control "AC-2(1)"

# Search (FTS5 or vector, depending on binary variant)
fisma-ref-mcp search "encryption at rest"
fisma-ref-mcp search "audit log retention" --limit 5

# Start the HTTP MCP server
fisma-ref-mcp serve --port 8080

# Start the stdio MCP server
fisma-ref-mcp serve --stdio
```

---

## How this helps a platform team meet FISMA High

### The core problem it solves

When developers and architects work with AI assistance on a FISMA High system, the AI draws on training data — which paraphrases controls, misremembers parameters, and conflates Low/Moderate/High requirements. Every inaccuracy in a design doc, PR description, or SSP narrative is a finding waiting to happen.

This tool gives your AI direct access to the official control text during development, so citations are exact and traceable back to the source document.

### Specific use cases

**Control-aware development**  
Developers ask the AI to implement a feature (session management, audit logging, access revocation) and the AI queries this server to retrieve the relevant controls before responding. Implementation decisions are made against the actual requirement, not a recollection of it. Control IDs appear in PR descriptions, ADRs, and commit messages — creating a continuous audit trail from code to requirement.

**SSP drafting**  
An ATO at High impact requires a System Security Plan that describes how each of the ~900 applicable High baseline controls is implemented. AI-assisted drafting with this tool produces narratives that quote the control verbatim, apply the correct parameters, and use the right control identifiers — rather than generating plausible-sounding text that an assessor has to verify against the source.

**Control responsibility partitioning**  
Platform teams own a specific slice of the control portfolio and provide inherited controls to hosted application teams. This tool lets you systematically enumerate your scope by family (typically AC, AU, SC, CM, SI, and parts of IA and PE) and produce a structured responsibility matrix. The AI can help reason about which controls are fully inherited from your cloud provider, which you implement at the platform layer, and which are shared with application teams.

**Assessment preparation**  
FISMA High assessments require evidence for every control in scope. The AI can generate control-specific assessment questions, interview guides, and evidence checklists by querying the examine/interview/test elements of each control — keyed to the correct control identifiers so findings map unambiguously to the SSP.

**Continuous monitoring**  
Annual assessments and continuous monitoring require ongoing evidence collection tied to specific controls. When your tooling references controls by ID throughout the development lifecycle, POA&M entries, deviation requests, and monitoring reports stay aligned to the right control without manual reconciliation.

### Limitations

**It does not implement compliance.** The tool is a reference. Your team still has to build, configure, and operate the controls. An AI that knows what SC-28 requires for protection of information at rest still needs you to actually encrypt the database.

**It does not cover the High baseline selection.** NIST SP 800-53B defines which controls are required at Low, Moderate, and High impact levels. That baseline data is not in this dataset. The tool can retrieve any control, but it does not know which controls are mandatory for your impact level.

**It does not include FedRAMP parameter overlays.** FedRAMP High tightens specific parameters on top of SP 800-53 (e.g., AU-11 mandates one year of audit record retention; the base control leaves the period organization-defined). Those overlays are not present in this dataset.

**It does not generate evidence.** Assessors require artifacts — configuration exports, screenshots, log samples, scan results. This tool helps you describe controls accurately; it does not produce the artifacts that demonstrate implementation.

**Assessment requires human judgment.** A 3PAO reviewing your SSP evaluates whether your described implementation satisfies the intent of the control. That determination is not automatable.

---

## Development

### Project structure

```
cmd/                  CLI commands (serve, search, control, family)
internal/
  nist/               NIST data types, JSON parsing, embed directives
    data/             nist-800-53r5.json + per-model vector index subdirectories
      nomic/          chromem.db + chromem-meta.json (nomic-embed-text:v1.5)
      qwen3/          chromem.db + chromem-meta.json (qwen3-embedding:4b)
      openai-small/   chromem.db + chromem-meta.json (text-embedding-3-small)
  store/              Unified data access (SQLite + chromem-go)
  mcp/                MCP server, tool registration, handlers
tools/
  gen-embeddings/     Build-time embedding generator (go run only)
```

### Build targets

```bash
make build               # compile with default (untagged) vector index
make build-nomic         # compile with nomic-embed-text:v1.5 index
make build-qwen3         # compile with qwen3-embedding:4b index
make build-openai-small  # compile with text-embedding-3-small index
make build-slim          # compile without vector index (FTS5 only)
```

### Regenerating vector indexes

Each embedding model has its own subdirectory under `internal/nist/data/`. Run the relevant target, then commit the updated files. CI embeds whichever files are committed — no API keys or Ollama required during the build.

```bash
make embed-nomic                          # requires Ollama + nomic-embed-text:v1.5
make embed-qwen3                          # requires Ollama + qwen3-embedding:4b
OPENAI_API_KEY=sk-... make embed-openai-small
```

After running one or more of these:

```bash
git add internal/nist/data/
git commit -m "update vector indexes"
git push
```

GitHub CI will then produce release binaries for all four variants automatically.

### How embedding detection works

Each binary embeds a `chromem-meta.json` alongside its vector index. At startup, `store.New()` reads the meta and automatically sets the embedding provider and model — so users never need to pass `--embedding-provider` or `--embedding-model` flags. If the provider requires an API key (OpenAI), the binary will prompt for `OPENAI_API_KEY` but nothing else.

Vectors from different embedding models are mathematically incompatible. The binary validates at startup that the runtime provider and model match the index — and exits with a clear error if they differ rather than returning silently wrong results.

### Running locally

```bash
go run . serve --stdio          # stdio MCP server
go run . family AC              # quick CLI lookup
go run . search "least privilege"
```

### After forglet synth

`go.mod` is managed by [forglet](https://github.com/forgant-foundry/forglet). After running `forglet synth`, restore the dependency list:

```bash
chmod 644 go.mod
go mod tidy
```

---

## License

See [LICENSE](LICENSE).
