# rel_store — relational data layer

An in-memory SQLite database (schema in `data/schema.sql`) populated at startup from the embedded JSON corpora. It provides deterministic lookups and FTS5 full-text search across all four corpora.

## Corpora and their tables

### NIST SP 800-53

| Table | Purpose |
|---|---|
| `families` | The 20 control families (AC, AU, CA, …) |
| `controls` | All 1,196 controls and enhancements |
| `control_baselines` | SP 800-53B baseline membership (Low / Moderate / High / Privacy) |
| `controls_fts` | FTS5 index over title, statement, discussion |

`controls.is_enhancement = 1` for control enhancements (e.g. AC-2(1)); `parent_id` points to the base control. Base controls have `is_enhancement = 0` and `parent_id = NULL`.

### FY 2025 IG FISMA Metrics

| Table | Purpose |
|---|---|
| `fisma_metrics` | The 35 metrics (question text and review cycle) |
| `fisma_maturity_levels` | Five maturity levels per metric with description, evidence, and assessor notes |
| `fisma_criteria` | Cross-references from each metric to SP 800-53 controls and/or CSF 2.0 subcategories |
| `fisma_metrics_fts` | FTS5 index over domain and question |

`fisma_criteria.ref_type` is `nist_800_53` or `nist_csf`. `control_id` and `csf_subcategory_id` are populated accordingly; the other column is NULL.

### NIST CSF 2.0

| Table | Purpose |
|---|---|
| `csf_functions` | The 6 functions (GV, ID, PR, DE, RS, RC) |
| `csf_categories` | Categories within each function |
| `csf_subcategories` | The 185 subcategories |
| `csf_examples` | Implementation examples for each subcategory (one row per example) |
| `csf_controls` | Official crosswalk: CSF subcategory → SP 800-53 control |
| `csf_subcategories_fts` | FTS5 index over subcategory text |

`csf_subcategories` carries a denormalised `function_id` in addition to `category_id` to make function-scoped queries cheaper.

### FedRAMP 20x

| Table | Purpose |
|---|---|
| `fedramp_terms` | 49 glossary terms with definitions |
| `ksi_themes` | 11 KSI themes |
| `ksi_indicators` | 60 KSI indicators, each belonging to a theme |
| `ksi_controls` | KSI indicator → SP 800-53 control references |
| `fedramp_requirements` | 163 MUST/SHOULD/MAY process requirements |
| `ksi_indicators_fts` | FTS5 index over KSI name and statement |
| `fedramp_requirements_fts` | FTS5 index over requirement name and statement |
| `fedramp_terms_fts` | FTS5 index over term and definition |

`fedramp_requirements.keyword` is `MUST`, `SHOULD`, or `MAY`. `version` is `rev5`, `20x`, or `both`.

## Foreign keys

```
families
  └── controls.family_id → families.id
        └── controls.parent_id → controls.id   (self; enhancements → base control)

fisma_metrics
  ├── fisma_maturity_levels.metric_id → fisma_metrics.id
  └── fisma_criteria.metric_id        → fisma_metrics.id

csf_functions
  └── csf_categories.function_id → csf_functions.id
        └── csf_subcategories.category_id → csf_categories.id
              └── csf_examples.subcategory_id → csf_subcategories.id

ksi_themes
  └── ksi_indicators.theme_id → ksi_themes.id
```

## Cross-corpus links

These columns reference rows in another corpus's tables but are stored as plain text (no SQLite FOREIGN KEY constraint, since the referenced table belongs to a different logical corpus):

| Column | References |
|---|---|
| `control_baselines.control_id` | `controls.id` |
| `fisma_criteria.control_id` | `controls.id` |
| `fisma_criteria.csf_subcategory_id` | `csf_subcategories.id` |
| `csf_controls.control_id` | `controls.id` |
| `ksi_controls.control_id` | `controls.id` |

These links are how the API surfaces cross-corpus queries: `get_metrics_by_control`, `get_ksis_by_control`, `get_csf_subcategory` (with control refs), and the CSF↔FISMA criteria join all resolve through these columns.
