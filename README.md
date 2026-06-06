# fisma-ref-mcp

A local [Model Context Protocol (MCP)](https://modelcontextprotocol.io) server that gives AI assistants direct, citeable access to five federal cybersecurity reference corpora: **NIST SP 800-53 Rev 5** security and privacy controls, **NIST SP 800-53B** impact baselines, **NIST Cybersecurity Framework 2.0**, the **FY 2025 IG FISMA Evaluation Metrics**, and the **FedRAMP Machine-Readable Requirements** (FRMR).

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

All corpora are embedded verbatim at build time from the source files listed below. Version identifiers are taken directly from the embedded JSON metadata — they reflect the exact document revision ingested, not a general edition label.

### NIST SP 800-53 Rev 5

| Field | Value |
|---|---|
| Document | Security and Privacy Controls for Information Systems and Organizations |
| Identifier | `SP_800_53_5_2_0` |
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

### NIST SP 800-53B

| Field | Value |
|---|---|
| Document | Control Baselines for Information Systems and Organizations |
| Identifier | `SP_800_53_B_5_2_0` |
| Version | **5.2.0** |
| Source | [nvlpubs.nist.gov](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-53B.pdf) |
| Low baseline | 149 controls/enhancements |
| Moderate baseline | 287 controls/enhancements |
| High baseline | 370 controls/enhancements |
| Privacy baseline | 96 controls/enhancements |

Baseline membership is surfaced on every control returned by `get_control` and `get_baseline`. A control's `baselines` field lists which of the four profiles it belongs to — enabling questions like "which High baseline controls do we not have coverage for?"

### NIST Cybersecurity Framework 2.0

| Field | Value |
|---|---|
| Document | Cybersecurity Framework |
| Identifier | `CSF_2_0_0` |
| Version | **2.0** |
| Source | [nist.gov/cyberframework](https://www.nist.gov/cyberframework) |
| Functions | 6 (Govern, Identify, Protect, Detect, Respond, Recover) |
| Categories | 45 |
| Subcategories | **185** |

Each subcategory includes its outcome statement and implementation examples. CSF 2.0 subcategory IDs (e.g. `GV.OC-01`, `PR.AA-03`) appear as criteria references in the FISMA metrics, making the corpora navigable together.

### FY 2025 IG FISMA Metrics

| Field | Value |
|---|---|
| Document | FY 2025 Inspector General FISMA Reporting Metrics |
| Version | **FY 2025** (published May 5, 2025) |
| Source | CISA / CIGIE, in coordination with OMB and DHS |
| Total metrics | **35** |
| Core metrics | 20 (assessed annually) |
| Supplemental metrics | 5 (new for FY 2025 — Zero Trust Architecture focus) |
| Maturity levels | Ad Hoc (1) through Optimized (5) |

Each metric includes the evaluation question, full maturity level descriptions, expected evidence, assessor best practices per level, and criteria references (NIST SP 800-53 control IDs, OMB guidance, and NIST publications where applicable).

FISMA domains covered: Identity Management and Access Control · Configuration Management · Data Protection and Privacy · Respond · Recover · Identify · Protect · Detect · Govern

### FedRAMP Machine-Readable Requirements (FRMR)

| Field | Value |
|---|---|
| Document | FedRAMP Machine-Readable Documentation |
| Version | **0.9.43-beta** |
| Last updated | 2026-04-08 |
| Source | fedramp.gov |
| Glossary terms (FRD) | 49 |
| KSI themes | 11 |
| KSI indicators | **60** |
| Process requirement categories (FRR) | 11 |
| Process requirements | **163** |

The FRMR captures the FedRAMP 20x framework — a shift from the traditional Low/Moderate/High baseline model to outcome-based **Key Security Indicators (KSIs)**. Each KSI indicator carries explicit SP 800-53 control references, completing the cross-corpus traceability chain: `get_ksis_by_control` shows which FedRAMP 20x outcomes a given control contributes to alongside the existing FISMA metric linkage.

FRR process requirement categories: ADS · CCM · FSI · ICP · MAS · PVA · SCG · SCN · UCM · VDR · KSI (each with rev5, 20x, or both applicability)

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

Once connected, the AI has access to seventeen tools.

### `search`

Semantic search across all indexed documents — NIST SP 800-53 controls, FY 2025 IG FISMA metrics, NIST CSF 2.0 subcategories, and FedRAMP 20x KSI indicators, process requirements, and glossary terms — in a single ranked result set. Each result includes a `source` field for provenance. Uses vector search when the binary includes a pre-built index; falls back to FTS5 otherwise.

```
query   string  (required) Natural-language description of what you are looking for
limit   number  (optional) Max results, default 10, max 50
source  string  (optional) Restrict to one corpus: "nist_800_53", "fisma_fy2025", "nist_csf_v2", or "fedramp_20x"
family  string  (optional) Restrict NIST SP 800-53 results to a specific family, e.g. "AC"
```

**Example prompts:**
- *"What covers multi-factor authentication across NIST controls, FISMA metrics, and FedRAMP KSIs?"*
- *"Search for incident response requirements"*
- *"Find FedRAMP 20x requirements for vulnerability detection"*
- *"What does CSF 2.0 say about organizational risk profiles?"*

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

### `get_baseline`

Returns all NIST SP 800-53 controls and enhancements included in a given SP 800-53B impact baseline. Each control includes its `baselines` field showing all profiles it belongs to.

```
baseline  string  (required) "low", "moderate", "high", or "privacy"
```

**Example prompts:**
- *"List all controls in the High baseline"*
- *"What's the difference in scope between the Moderate and High baselines?"*
- *"Which AC family controls are required at Low impact?"*

### `get_metrics_by_control`

Returns all FY 2025 IG FISMA metrics that reference a given NIST SP 800-53 control. Use this to understand the FISMA audit impact of implementing or failing a specific control.

```
control_id  string  (required) NIST SP 800-53 control ID, e.g. "AC-2" or "SI-3"
```

**Example prompts:**
- *"Which FISMA metrics reference AC-2?"*
- *"If we implement IA-5, which IG metrics does that contribute evidence toward?"*
- *"What's the FISMA exposure if we have a weakness in CM-6?"*

### `get_csf_subcategories_by_control`

Returns all NIST CSF 2.0 subcategories that map to a given SP 800-53 control via the official NIST crosswalk. Useful for understanding a control's CSF placement or generating framework-aligned evidence narratives.

```
control_id  string  (required) NIST SP 800-53 control ID, e.g. "AC-2" or "IA-5"
```

**Example prompts:**
- *"Which CSF 2.0 subcategories map to AC-17?"*
- *"What CSF outcomes does implementing IA-5 contribute toward?"*

### `get_metrics_by_csf_subcategory`

Returns all FY 2025 IG FISMA metrics that reference a given CSF 2.0 subcategory ID. Completes the three-way traceability chain: control → CSF subcategory → FISMA metric.

```
subcategory_id  string  (required) CSF 2.0 subcategory ID, e.g. "GV.OC-01" or "PR.AA-03"
```

**Example prompts:**
- *"Which FISMA metrics reference PR.AA-03?"*
- *"What IG audit metrics are tied to the GV.SC subcategories?"*

### `list_csf_functions`

Returns NIST CSF 2.0 functions with their categories. Optionally filter to a single function.

```
function  string  (optional) Function ID to filter to, e.g. "GV" for Govern
```

**Example prompts:**
- *"List all CSF 2.0 functions and their categories"*
- *"What categories are in the Protect function?"*

### `get_csf_subcategory`

Returns a single NIST CSF 2.0 subcategory by its identifier, including the outcome statement and implementation examples.

```
id  string  (required) Subcategory identifier, e.g. "GV.OC-01" or "PR.AA-03"
```

**Example prompts:**
- *"Get me the full text of GV.OC-01"*
- *"What does PR.AA-03 require and what are the implementation examples?"*

### `list_ksi_themes`

Returns all FedRAMP 20x Key Security Indicator themes with their indicators. Each indicator includes its outcome statement and the SP 800-53 controls it references. Optionally filter to a single theme.

```
theme  string  (optional) Theme short name, e.g. "IAM", "MLA", "SVC", "CNA"
```

**Example prompts:**
- *"List all FedRAMP 20x KSI themes and their indicators"*
- *"What KSIs are in the Identity and Access Management theme?"*

### `get_ksi`

Returns a single FedRAMP 20x KSI indicator by its ID, including its outcome statement and referenced SP 800-53 controls.

```
id  string  (required) KSI indicator ID, e.g. "KSI-IAM-MFA" or "KSI-MLA-ALA"
```

**Example prompts:**
- *"Get me the full details of KSI-IAM-MFA"*
- *"What SP 800-53 controls does KSI-CNA-IBP reference?"*

### `get_ksis_by_control`

Returns all FedRAMP 20x KSI indicators that reference a given NIST SP 800-53 control. Use this to understand which FedRAMP outcome-based security indicators a control implementation contributes toward.

```
control_id  string  (required) NIST SP 800-53 control identifier, e.g. "IA-5" or "AC-2"
```

**Example prompts:**
- *"Which FedRAMP KSIs reference IA-5?"*
- *"If we implement SC-28, which FedRAMP 20x outcomes does that support?"*

### `list_fedramp_requirements`

Returns FedRAMP process requirements (MUST/SHOULD statements) from the FRR section. Filter by category and/or version path.

```
category  string  (optional) Requirement category, e.g. "VDR", "SCN", "ADS", "CCM"
version   string  (optional) FedRAMP path: "rev5", "20x" (also returns "both" requirements)
```

**Example prompts:**
- *"List all FedRAMP MUST requirements for vulnerability detection"*
- *"What does FedRAMP 20x require for significant change notification?"*

### `get_fedramp_term`

Returns a single FedRAMP glossary term definition by its ID.

```
id  string  (required) FedRAMP term ID, e.g. "FRD-ACV" (Accepted Vulnerability)
```

**Example prompts:**
- *"What is the FedRAMP definition of an accepted vulnerability?"*
- *"Define FRD-PER"*

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
fisma-ref-mcp search "organizational risk profile" --source nist_csf_v2
fisma-ref-mcp search "phishing resistant authentication" --source fedramp_20x

# Start the HTTP MCP server
fisma-ref-mcp serve --port 8080

# Start the stdio MCP server
fisma-ref-mcp serve --stdio
```

---

## Limitations

**It does not implement compliance.** The tool is a reference. Your team still has to build, configure, and operate the controls. An AI that knows what SC-28 requires for protection of information at rest still needs you to actually encrypt the database.

**The FedRAMP FRMR is a beta document.** The embedded version is `0.9.43-beta` (2026-04-08). The FedRAMP 20x framework is still evolving; KSI indicators and process requirements may change before final release.

**FedRAMP parameter overlays are not included.** The FRMR covers outcome-based KSIs and process requirements. The traditional FedRAMP control parameter values (e.g., AU-11 log retention = 1 year) from the Rev 5 baseline profiles are not in this dataset.

**It does not generate evidence.** Assessors require artifacts — configuration exports, screenshots, log samples, scan results. This tool helps you describe controls and maturity levels accurately; it does not produce the artifacts that demonstrate implementation.

**Assessment requires human judgment.** A 3PAO or IG reviewing your implementation evaluates whether it satisfies the intent of the control. That determination is not automatable.

---

## Development

### Project structure

```
cmd/                    CLI commands (serve, search, control, family)
internal/
  nist_800_53/          SP 800-53 5.2.0 + SP 800-53B 5.2.0 data types, JSON parsing, embed
    data/               nist-800-53r5.json (SP_800_53_5_2_0), nist-800-53b.json (SP_800_53_B_5_2_0)
  fedramp/              FedRAMP FRMR 0.9.43-beta types, JSON parsing, embed
    data/               FRMR.documentation.json
  nist_csf/             NIST CSF 2.0 (CSF_2_0_0) data types, JSON parsing, embed
    data/               nist-csf-2.0.json
  fisma/                FY 2025 IG FISMA data types, JSON parsing, embed
    data/               fy2025-ig-fisma-metrics.json + context markdown
  vec_store/            Vector index: VectorMeta, PrebuiltVector(), build-tag embed files
                        vector.go/vector_stub.go — chromem-go index and query
                        documents.go — exported document builders (shared with gen-embeddings)
    data/
      nomic/            chromem.db + chromem-meta.json (nomic-embed-text:v1.5)
      qwen3/            chromem.db + chromem-meta.json (qwen3-embedding:4b)
      openai-small/     chromem.db + chromem-meta.json (text-embedding-3-small)
  rel_store/            Unified data access layer — see internal/rel_store/README.md
    data/schema.sql     SQLite schema DDL (//go:embed)
  mcp/                  MCP server, tool registration, handlers
tools/
  gen-embeddings/       Build-time embedding generator (indexes all four searchable corpora)
  parse-fisma-metrics/  PDF parser for the IG FISMA metrics document
```

The relational database schema, table descriptions, and cross-corpus foreign key relationships are documented in [internal/rel_store/README.md](internal/rel_store/README.md).

### Build targets

```bash
make build-nomic         # compile with nomic-embed-text:v1.5 index
make build-qwen3         # compile with qwen3-embedding:4b index
make build-openai-small  # compile with text-embedding-3-small index
make build-slim          # compile without vector index (FTS5 only)
make build-all           # build all four variants to named binaries
```

### Regenerating vector indexes

Each embedding model has its own subdirectory under `internal/vec_store/data/`. The index covers all four searchable corpora: NIST SP 800-53 controls, FY 2025 IG FISMA metrics, NIST CSF 2.0 subcategories, and FedRAMP 20x KSI indicators, process requirements, and glossary terms. Run the relevant target, then commit the updated files. CI embeds whichever files are committed — no API keys or Ollama required during the build.

```bash
make embed-nomic                          # requires Ollama + nomic-embed-text:v1.5
make embed-qwen3                          # requires Ollama + qwen3-embedding:4b
OPENAI_API_KEY=sk-... make embed-openai-small
make embed-all                            # regenerate all three (requires both)
```

After running:

```bash
git add internal/vec_store/data/
git commit -m "update vector indexes"
git push
```

GitHub CI will then produce release binaries for all four variants automatically.

### How embedding detection works

Each binary embeds a `chromem-meta.json` alongside its vector index. At startup, `store.New()` reads the meta and automatically sets the embedding provider and model — so users never need to pass `--embedding-provider` or `--embedding-model` flags. If the provider requires an API key (OpenAI), the binary will look for `OPENAI_API_KEY` in the environment.

Vectors from different embedding models are mathematically incompatible. The binary validates at startup that the runtime provider and model match the index — and exits with a clear error if they differ rather than returning silently wrong results.

### Running locally

```bash
go run . serve --stdio                                        # stdio MCP server
go run . family AC                                            # quick NIST lookup
go run . search "least privilege"                             # cross-corpus search
go run . search "maturity level" --source fisma_fy2025
go run . search "organizational risk" --source nist_csf_v2
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
