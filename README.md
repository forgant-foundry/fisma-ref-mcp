# fisma-ref-mcp

A local [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server that gives AI assistants direct, citeable access to two federal cybersecurity reference corpora: **NIST SP 800-53 Rev 5** security and privacy controls and the **FY 2025 IG FISMA Evaluation Metrics**.

Install it once, point your AI at it, and every suggestion about access control, audit logging, encryption, or identity comes with a traceable reference to the exact control or metric that requires it — not a paraphrase, the official text.

---

## Why this exists

AI assistants are useful for compliance work — drafting SSP narratives, reviewing architecture decisions, writing policy, generating audit checklists, mapping control implementations to FISMA maturity levels. The problem is that they paraphrase controls and metrics from training data, which gets the spirit right and the specifics wrong. Wrong control IDs, wrong maturity level descriptions, wrong evidence requirements. An assessor reviewing your SSP or IG FISMA submission will find every one of those mistakes.

This tool eliminates that problem by embedding both corpora directly in the binary and exposing them to AI tools via MCP. The AI queries the actual source text; every citation is exact. And because both corpora are indexed together, the AI can reason across them — seeing which NIST controls feed which FISMA metrics, and which metrics are relevant to a given domain.

---

## Who is this for

**Software developers** working on federal systems or FedRAMP products can ask their AI assistant "what controls apply here?" and get the exact NIST control text inline — not a paraphrase, the authoritative source. The `get_metrics_by_control` tool closes the loop the other direction: implement AC-2 and immediately see which of the 35 IG FISMA metrics you're contributing evidence toward. Traceability from code change to compliance posture, without leaving the editor.

**Auditors and IGs** can query maturity level descriptions and assessor notes directly instead of navigating a 200-page PDF. Pull a specific FISMA metric and see what "Consistently Implemented" looks like vs. "Managed and Measurable" — the description, expected evidence, and assessor notes side by side. When an agency claims a control is implemented, cross-check which FISMA metrics that control feeds and whether their evidence package covers the right dimensions.

**CTOs and CISOs** get the domain-level rollup: `list_fisma_metrics` grouped by domain maps where gaps cluster across Identity Management, Configuration Management, Incident Response, and the other FISMA domains. Paired with an AI agent, the question "given our current control coverage, which maturity levels are we likely missing evidence for?" produces a prioritized gap list rather than a spreadsheet exercise — making the connection between security investment and IG audit outcomes legible at the executive level.

The shared thread: the MCP server makes both corpora machine-readable to AI, so each role gets the right slice surfaced in their own workflow rather than navigating the same documents repeatedly.

---

## The NIST ↔ FISMA connection

NIST SP 800-53 controls and FISMA IG metrics are different slices of the same requirement. Controls define *what* must be implemented; FISMA metrics define *how well* those implementations are evaluated. The two are tightly coupled — most FISMA metrics reference specific SP 800-53 controls in their criteria, and most controls map to at least one FISMA domain.

This server makes that relationship queryable. Some examples of what becomes possible when both corpora are available together:

**Forward traceability — from implementation to audit outcome**
> *"We're implementing AC-2. Which FISMA metrics does that contribute evidence toward, and what does the Managed and Measurable level expect?"*

The `get_metrics_by_control` tool returns every metric that references AC-2, with full maturity level descriptions. A team can see exactly what evidence they need to produce, not just which controls to implement.

**Reverse traceability — from IG finding to root cause controls**
> *"Our agency scored Level 3 on the Identity Management metrics. What NIST controls are likely driving that gap?"*

Search the FISMA metrics for the relevant domain, pull the criteria references for the maturity level you're trying to reach, then retrieve the full control text to understand what implementation actually requires.

**Gap analysis by FISMA domain**
> *"Which FISMA domains map to the Access Control family, and what does Managed and Measurable require for each metric in those domains?"*

Combine `list_fisma_metrics` filtered by domain with `get_family` to see which controls underpin the domain and where evidence gaps are likely to appear.

**Evidence matrix generation**
> *"Given our FISMA metrics for Configuration Management, generate an evidence collection checklist that maps each maturity level to the relevant SP 800-53 controls."*

An AI with both corpora available can synthesize a structured evidence matrix — control by control, metric by metric — rather than requiring a human to cross-reference two documents manually.

**Authorization package alignment**
> *"Review our SSP narrative for CM-6. Does it address what the FISMA metrics for Configuration Management look for at Level 4?"*

Developers and compliance staff can verify that SSP narratives are written to address both the control requirement and the assessment criteria before submission.

---

## Included data

### NIST SP 800-53 Rev 5

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

> **Note:** This dataset reflects **SP 800-53 5.2.0**. It does not include NIST SP 800-53B baseline designations (Low/Moderate/High impact level assignments) or FedRAMP-specific parameter overlays. See [Limitations](#limitations) for details.

### FY 2025 IG FISMA Metrics

| Field | Value |
|---|---|
| Document | FY 2025 Inspector General FISMA Reporting Metrics |
| Source | CISA / CIGIE, in coordination with OMB and DHS |
| Total metrics | **35** |
| Core metrics | 20 (assessed annually) |
| Supplemental metrics | 5 (new for FY 2025 — Zero Trust Architecture focus) |
| Maturity levels | Ad Hoc (1) through Optimized (5) |

Each metric includes the evaluation question, full maturity level descriptions, expected evidence, assessor best practices per level, and criteria references (NIST SP 800-53 control IDs, OMB guidance, and NIST publications where applicable).

FISMA domains covered: Identity Management and Access Control · Configuration Management · Data Protection and Privacy · Respond · Recover · Identify · Protect · Detect · Govern

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

**Vector search** uses pre-built semantic embeddings to find documents by meaning, even when exact words don't appear in the source text. **FTS5** is full-text keyword search with BM25 relevance ranking — fast, offline, and no configuration required. Both search modes cover NIST controls and FISMA metrics simultaneously.

If you don't have Ollama and don't want to manage an OpenAI key, the slim build is fully self-contained and works well for most lookups.

### Build from source

Requires Go 1.25+.

```bash
git clone https://github.com/forgant-foundry/fisma-ref-mcp
cd fisma-ref-mcp
make build-nomic         # compile with nomic-embed-text:v1.5 vector index
make build-qwen3         # compile with qwen3-embedding:4b vector index
make build-openai-small  # compile with text-embedding-3-small vector index
make build-slim          # compile without vector index (FTS5 only)
make build-all           # build all four variants to named binaries
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

**slim variant** (no configuration needed):

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

Once connected, the AI has access to seven tools.

### `search`

Semantic search across all indexed documents — NIST SP 800-53 controls and FY 2025 IG FISMA metrics — in a single ranked result set. Each result includes a `source` field (`nist_800_53` or `fisma_fy2025`) for provenance. Uses vector search when the binary includes a pre-built index; falls back to FTS5 otherwise.

```
query   string  (required) Natural-language description of what you are looking for
limit   number  (optional) Max results, default 10, max 50
source  string  (optional) Restrict to "nist_800_53" or "fisma_fy2025"
family  string  (optional) Restrict NIST results to a specific family, e.g. "AC"
```

**Example prompts:**
- *"What covers multi-factor authentication across both NIST controls and FISMA metrics?"*
- *"Search for incident response requirements"*
- *"Find FISMA metrics related to zero trust"*

### `get_control`

Retrieve the full text of a single NIST SP 800-53 control by ID.

```
id  string  (required) Control ID, e.g. "AC-1", "SI-3", "AC-2(1)"
```

**Example prompts:**
- *"Get me the full text of AC-17"*
- *"What does SI-7(1) say?"*

### `list_families`

Returns all 20 NIST SP 800-53 control families with their IDs and titles. Useful for exploration and orientation.

### `get_family`

Returns all base controls (no enhancements) in a given NIST family.

```
id  string  (required) Two-letter family ID, e.g. "AC", "SI"
```

**Example prompts:**
- *"What controls are in the IA family?"*
- *"List all audit controls"*

### `list_fisma_metrics`

Returns FY 2025 IG FISMA evaluation metrics with their domain, question, and review cycle. Optionally filter by domain.

```
domain  string  (optional) Filter to a specific domain, e.g. "Identity Management and Access Control"
```

**Example prompts:**
- *"List all FISMA metrics for Configuration Management"*
- *"What are all the supplemental Zero Trust metrics?"*

### `get_fisma_metric`

Returns a single FISMA metric by ID, including the full maturity level descriptions, expected evidence, assessor best practices per level, and all criteria references (NIST SP 800-53 control IDs and other applicable guidance).

```
id  number  (required) Metric ID, 1–35
```

**Example prompts:**
- *"Get me the full details of FISMA metric 19"*
- *"What does metric 7 require at the Managed and Measurable level?"*
- *"What evidence do assessors look for in metric 3 at Level 4?"*

### `get_metrics_by_control`

Returns all FY 2025 IG FISMA metrics that reference a given NIST SP 800-53 control. Use this to understand the FISMA audit impact of implementing or failing a specific control.

```
control_id  string  (required) NIST SP 800-53 control ID, e.g. "AC-2" or "SI-3"
```

**Example prompts:**
- *"Which FISMA metrics reference AC-2?"*
- *"If we implement IA-5, which IG metrics does that contribute evidence toward?"*
- *"What's the FISMA exposure if we have a weakness in CM-6?"*

---

## CLI usage

Every MCP tool is also available as a standalone command that prints JSON to stdout — useful for scripting, piping into `jq`, or quick lookups without starting a server.

```bash
# List all NIST control families
fisma-ref-mcp family

# List all controls in a family
fisma-ref-mcp family AC

# Get a single control (case-insensitive)
fisma-ref-mcp control AC-1
fisma-ref-mcp control ac-17
fisma-ref-mcp control "AC-2(1)"

# Search across both corpora
fisma-ref-mcp search "encryption at rest"
fisma-ref-mcp search "identity governance maturity" --limit 5

# Search within a specific corpus
fisma-ref-mcp search "multi-factor authentication" --source nist_800_53
fisma-ref-mcp search "zero trust" --source fisma_fy2025

# Start the HTTP MCP server
fisma-ref-mcp serve --port 8080

# Start the stdio MCP server
fisma-ref-mcp serve --stdio
```

---

## Limitations

**It does not implement compliance.** The tool is a reference. Your team still has to build, configure, and operate the controls. An AI that knows what SC-28 requires for protection of information at rest still needs you to actually encrypt the database.

**It does not cover baseline selection.** NIST SP 800-53B defines which controls are required at Low, Moderate, and High impact levels. That baseline data is not in this dataset. The tool can retrieve any control, but it does not know which controls are mandatory for your impact level.

**It does not include FedRAMP parameter overlays.** FedRAMP High tightens specific parameters on top of SP 800-53 (e.g., AU-11 mandates one year of audit record retention; the base control leaves the period organization-defined). Those overlays are not present in this dataset.

**It does not generate evidence.** Assessors require artifacts — configuration exports, screenshots, log samples, scan results. This tool helps you describe controls and maturity levels accurately; it does not produce the artifacts that demonstrate implementation.

**Assessment requires human judgment.** A 3PAO or IG reviewing your implementation evaluates whether it satisfies the intent of the control. That determination is not automatable.

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
  fisma/              FY 2025 IG FISMA data types, JSON parsing, embed directives
    data/             fy2025-ig-fisma-metrics.json + context markdown
  store/              Unified data access layer (SQLite + chromem-go)
  mcp/                MCP server, tool registration, handlers
tools/
  gen-embeddings/     Build-time embedding generator (indexes both corpora)
  parse-fisma-metrics/  PDF parser for the IG FISMA metrics document
```

### Build targets

```bash
make build-nomic         # compile with nomic-embed-text:v1.5 index
make build-qwen3         # compile with qwen3-embedding:4b index
make build-openai-small  # compile with text-embedding-3-small index
make build-slim          # compile without vector index (FTS5 only)
make build-all           # build all four variants to named binaries
```

### Regenerating vector indexes

Each embedding model has its own subdirectory under `internal/nist/data/`. The index covers both NIST SP 800-53 controls and FISMA metrics. Run the relevant target, then commit the updated files. CI embeds whichever files are committed — no API keys or Ollama required during the build.

```bash
make embed-nomic                          # requires Ollama + nomic-embed-text:v1.5
make embed-qwen3                          # requires Ollama + qwen3-embedding:4b
OPENAI_API_KEY=sk-... make embed-openai-small
make embed-all                            # regenerate all three (requires both)
```

After running:

```bash
git add internal/nist/data/
git commit -m "update vector indexes"
git push
```

GitHub CI will then produce release binaries for all four variants automatically.

### How embedding detection works

Each binary embeds a `chromem-meta.json` alongside its vector index. At startup, `store.New()` reads the meta and automatically sets the embedding provider and model — so users never need to pass `--embedding-provider` or `--embedding-model` flags. If the provider requires an API key (OpenAI), the binary will look for `OPENAI_API_KEY` in the environment.

Vectors from different embedding models are mathematically incompatible. The binary validates at startup that the runtime provider and model match the index — and exits with a clear error if they differ rather than returning silently wrong results.

### Running locally

```bash
go run . serve --stdio                       # stdio MCP server
go run . family AC                           # quick NIST lookup
go run . search "least privilege"            # cross-corpus search
go run . search "maturity level" --source fisma_fy2025
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
