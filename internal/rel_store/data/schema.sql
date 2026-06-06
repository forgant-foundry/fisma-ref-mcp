CREATE TABLE families (
    id    TEXT PRIMARY KEY,
    title TEXT NOT NULL
);

CREATE TABLE controls (
    id             TEXT PRIMARY KEY,
    family_id      TEXT NOT NULL,
    title          TEXT NOT NULL,
    statement      TEXT NOT NULL DEFAULT '',
    discussion     TEXT NOT NULL DEFAULT '',
    is_enhancement INTEGER NOT NULL DEFAULT 0,
    parent_id      TEXT,
    FOREIGN KEY (family_id) REFERENCES families(id)
);

CREATE INDEX idx_controls_family ON controls(family_id);
CREATE INDEX idx_controls_parent ON controls(parent_id);

CREATE TABLE control_baselines (
    control_id TEXT NOT NULL,
    baseline   TEXT NOT NULL,
    PRIMARY KEY (control_id, baseline)
);
CREATE INDEX idx_control_baselines_ctrl ON control_baselines(control_id);
CREATE INDEX idx_control_baselines_bl   ON control_baselines(baseline);

CREATE VIRTUAL TABLE controls_fts USING fts5(
    id       UNINDEXED,
    title,
    statement,
    discussion,
    tokenize = 'unicode61'
);

CREATE TABLE fisma_metrics (
    id           INTEGER PRIMARY KEY,
    domain       TEXT    NOT NULL,
    question     TEXT    NOT NULL,
    review_cycle TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE fisma_maturity_levels (
    metric_id      INTEGER NOT NULL REFERENCES fisma_metrics(id),
    level          TEXT    NOT NULL,
    description    TEXT    NOT NULL DEFAULT '',
    evidence       TEXT    NOT NULL DEFAULT '',
    assessor_notes TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (metric_id, level)
);

CREATE TABLE fisma_criteria (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_id          INTEGER NOT NULL REFERENCES fisma_metrics(id),
    ref_type           TEXT    NOT NULL,
    ref_text           TEXT    NOT NULL DEFAULT '',
    control_id         TEXT,   -- populated when ref_type = 'nist_800_53'
    csf_subcategory_id TEXT    -- populated when ref_type = 'nist_csf'
);

CREATE INDEX idx_fisma_criteria_metric   ON fisma_criteria(metric_id);
CREATE INDEX idx_fisma_criteria_ctrl     ON fisma_criteria(control_id);
CREATE INDEX idx_fisma_criteria_csf      ON fisma_criteria(csf_subcategory_id);
CREATE INDEX idx_fisma_maturity_metric   ON fisma_maturity_levels(metric_id);

CREATE VIRTUAL TABLE fisma_metrics_fts USING fts5(
    id       UNINDEXED,
    domain,
    question,
    tokenize = 'unicode61'
);

CREATE TABLE csf_functions (
    id    TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    text  TEXT NOT NULL DEFAULT ''
);

CREATE TABLE csf_categories (
    id          TEXT PRIMARY KEY,
    function_id TEXT NOT NULL REFERENCES csf_functions(id),
    title       TEXT NOT NULL DEFAULT '',
    text        TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_csf_categories_function ON csf_categories(function_id);

CREATE TABLE csf_subcategories (
    id          TEXT PRIMARY KEY,
    category_id TEXT NOT NULL REFERENCES csf_categories(id),
    function_id TEXT NOT NULL REFERENCES csf_functions(id),
    text        TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_csf_subcategories_category ON csf_subcategories(category_id);
CREATE INDEX idx_csf_subcategories_function ON csf_subcategories(function_id);

CREATE TABLE csf_examples (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    subcategory_id TEXT NOT NULL REFERENCES csf_subcategories(id),
    text           TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_csf_examples_sub ON csf_examples(subcategory_id);

CREATE TABLE csf_controls (
    subcategory_id TEXT NOT NULL REFERENCES csf_subcategories(id),
    control_id     TEXT NOT NULL,
    PRIMARY KEY (subcategory_id, control_id)
);

CREATE INDEX idx_csf_controls_sub  ON csf_controls(subcategory_id);
CREATE INDEX idx_csf_controls_ctrl ON csf_controls(control_id);

CREATE VIRTUAL TABLE csf_subcategories_fts USING fts5(
    id          UNINDEXED,
    category_id UNINDEXED,
    function_id UNINDEXED,
    text,
    tokenize = 'unicode61'
);

CREATE TABLE fedramp_terms (
    id         TEXT PRIMARY KEY,
    term       TEXT NOT NULL,
    definition TEXT NOT NULL DEFAULT '',
    note       TEXT NOT NULL DEFAULT ''
);

CREATE TABLE ksi_themes (
    id         TEXT PRIMARY KEY,
    short_name TEXT NOT NULL,
    name       TEXT NOT NULL,
    theme      TEXT NOT NULL DEFAULT ''
);

CREATE TABLE ksi_indicators (
    id        TEXT PRIMARY KEY,
    theme_id  TEXT NOT NULL REFERENCES ksi_themes(id),
    name      TEXT NOT NULL,
    statement TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_ksi_indicators_theme ON ksi_indicators(theme_id);

CREATE TABLE ksi_controls (
    indicator_id TEXT NOT NULL REFERENCES ksi_indicators(id),
    control_id   TEXT NOT NULL,
    PRIMARY KEY (indicator_id, control_id)
);

CREATE INDEX idx_ksi_controls_ctrl ON ksi_controls(control_id);

CREATE TABLE fedramp_requirements (
    id        TEXT PRIMARY KEY,
    category  TEXT NOT NULL,
    name      TEXT NOT NULL DEFAULT '',
    statement TEXT NOT NULL DEFAULT '',
    keyword   TEXT NOT NULL DEFAULT '',
    version   TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_fedramp_req_category ON fedramp_requirements(category);

CREATE VIRTUAL TABLE ksi_indicators_fts USING fts5(
    id       UNINDEXED,
    theme_id UNINDEXED,
    name,
    statement,
    tokenize = 'unicode61'
);

CREATE VIRTUAL TABLE fedramp_requirements_fts USING fts5(
    id       UNINDEXED,
    category UNINDEXED,
    name,
    statement,
    tokenize = 'unicode61'
);

CREATE VIRTUAL TABLE fedramp_terms_fts USING fts5(
    id   UNINDEXED,
    term,
    definition,
    tokenize = 'unicode61'
);
