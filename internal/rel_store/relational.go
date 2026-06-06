package rel_store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/fedramp"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/fisma"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_800_53"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_csf"
	_ "modernc.org/sqlite"
)

type relationalDB struct {
	db *sql.DB
}

func newRelationalDB(families []nist_800_53.Family, controls []nist_800_53.Control, baselines map[string][]string, metrics []fisma.Metric, fns []nist_csf.Function, cats []nist_csf.Category, subs []nist_csf.Subcategory, frmr *fedramp.Catalog) (*relationalDB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := seed(db, families, controls); err != nil {
		db.Close()
		return nil, err
	}
	if err := seedBaselines(db, baselines); err != nil {
		db.Close()
		return nil, err
	}
	if err := seedFismaMetrics(db, metrics); err != nil {
		db.Close()
		return nil, err
	}
	if err := seedCSF(db, fns, cats, subs); err != nil {
		db.Close()
		return nil, err
	}
	if err := seedFedRAMP(db, frmr); err != nil {
		db.Close()
		return nil, err
	}
	return &relationalDB{db: db}, nil
}

func (r *relationalDB) close() error { return r.db.Close() }

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
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
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			metric_id  INTEGER NOT NULL REFERENCES fisma_metrics(id),
			ref_type   TEXT    NOT NULL,
			ref_text   TEXT    NOT NULL DEFAULT '',
			control_id TEXT             -- FK into controls.id when ref_type = 'nist_800_53'
		);

		CREATE INDEX idx_fisma_criteria_metric   ON fisma_criteria(metric_id);
		CREATE INDEX idx_fisma_criteria_ctrl     ON fisma_criteria(control_id);
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
			id       TEXT PRIMARY KEY,
			category TEXT NOT NULL,
			name     TEXT NOT NULL DEFAULT '',
			statement TEXT NOT NULL DEFAULT '',
			keyword  TEXT NOT NULL DEFAULT '',
			version  TEXT NOT NULL DEFAULT ''
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
	`)
	return err
}

func seed(db *sql.DB, families []nist_800_53.Family, controls []nist_800_53.Control) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, f := range families {
		if _, err := tx.Exec(
			`INSERT INTO families (id, title) VALUES (?, ?)`,
			f.ID, f.Title,
		); err != nil {
			return fmt.Errorf("insert family %s: %w", f.ID, err)
		}
	}

	for _, c := range controls {
		var parentID *string
		if c.ParentID != "" {
			parentID = &c.ParentID
		}
		enhancement := 0
		if c.IsEnhancement {
			enhancement = 1
		}
		if _, err := tx.Exec(
			`INSERT INTO controls (id, family_id, title, statement, discussion, is_enhancement, parent_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			c.ID, c.FamilyID, c.Title,
			c.Statement, c.Discussion,
			enhancement, parentID,
		); err != nil {
			return fmt.Errorf("insert control %s: %w", c.ID, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO controls_fts (id, title, statement, discussion) VALUES (?, ?, ?, ?)`,
			c.ID, c.Title, c.Statement, c.Discussion,
		); err != nil {
			return fmt.Errorf("insert fts %s: %w", c.ID, err)
		}
	}

	return tx.Commit()
}

func seedFismaMetrics(db *sql.DB, metrics []fisma.Metric) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin fisma tx: %w", err)
	}
	defer tx.Rollback()

	for _, m := range metrics {
		if _, err := tx.Exec(
			`INSERT INTO fisma_metrics (id, domain, question, review_cycle) VALUES (?, ?, ?, ?)`,
			m.ID, m.Domain, m.Question, m.ReviewCycle,
		); err != nil {
			return fmt.Errorf("insert metric %d: %w", m.ID, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO fisma_metrics_fts (id, domain, question) VALUES (?, ?, ?)`,
			m.ID, m.Domain, m.Question,
		); err != nil {
			return fmt.Errorf("insert metric fts %d: %w", m.ID, err)
		}

		for _, lvl := range m.MaturityLevels {
			if _, err := tx.Exec(
				`INSERT INTO fisma_maturity_levels (metric_id, level, description, evidence, assessor_notes)
				 VALUES (?, ?, ?, ?, ?)`,
				m.ID, lvl.Level, lvl.Description, lvl.Evidence, lvl.AssessorNotes,
			); err != nil {
				return fmt.Errorf("insert maturity level %d/%s: %w", m.ID, lvl.Level, err)
			}
		}

		for _, c := range m.Criteria {
			if len(c.ControlIDs) == 0 {
				// Stub record: no control linkage yet
				if _, err := tx.Exec(
					`INSERT INTO fisma_criteria (metric_id, ref_type, ref_text, control_id)
					 VALUES (?, ?, ?, NULL)`,
					m.ID, c.RefType, c.RefText,
				); err != nil {
					return fmt.Errorf("insert criterion metric %d: %w", m.ID, err)
				}
				continue
			}
			for _, ctrlID := range c.ControlIDs {
				if _, err := tx.Exec(
					`INSERT INTO fisma_criteria (metric_id, ref_type, ref_text, control_id)
					 VALUES (?, ?, ?, ?)`,
					m.ID, c.RefType, c.RefText, ctrlID,
				); err != nil {
					return fmt.Errorf("insert nist criterion metric %d ctrl %s: %w", m.ID, ctrlID, err)
				}
			}
		}
	}

	return tx.Commit()
}

// FismaMetric is the public type returned by FISMA metric queries.
type FismaMetric struct {
	ID           int
	Domain       string
	Question     string
	ReviewCycle  string
	MaturityLevels []FismaMaturityLevel
	Criteria     []FismaCriterion
}

// FismaMaturityLevel holds one maturity level record.
type FismaMaturityLevel struct {
	Level         string
	Description   string
	Evidence      string
	AssessorNotes string
}

// FismaCriterion holds one criteria reference.
type FismaCriterion struct {
	RefType   string
	RefText   string
	ControlID string // empty when ref_type != "nist_800_53"
}

func (r *relationalDB) getFismaMetric(ctx context.Context, id int) (*FismaMetric, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, domain, question, review_cycle FROM fisma_metrics WHERE id = ?`, id)
	var m FismaMetric
	if err := row.Scan(&m.ID, &m.Domain, &m.Question, &m.ReviewCycle); err == sql.ErrNoRows {
		return nil, fmt.Errorf("fisma metric %d not found", id)
	} else if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT level, description, evidence, assessor_notes
		 FROM fisma_maturity_levels WHERE metric_id = ? ORDER BY rowid`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var l FismaMaturityLevel
		if err := rows.Scan(&l.Level, &l.Description, &l.Evidence, &l.AssessorNotes); err != nil {
			return nil, err
		}
		m.MaturityLevels = append(m.MaturityLevels, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	crows, err := r.db.QueryContext(ctx,
		`SELECT ref_type, ref_text, COALESCE(control_id,'')
		 FROM fisma_criteria WHERE metric_id = ? ORDER BY rowid`, id)
	if err != nil {
		return nil, err
	}
	defer crows.Close()
	for crows.Next() {
		var c FismaCriterion
		if err := crows.Scan(&c.RefType, &c.RefText, &c.ControlID); err != nil {
			return nil, err
		}
		m.Criteria = append(m.Criteria, c)
	}
	return &m, crows.Err()
}

func (r *relationalDB) listFismaMetrics(ctx context.Context, domain string) ([]FismaMetric, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if domain == "" {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, domain, question, review_cycle FROM fisma_metrics ORDER BY id`)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, domain, question, review_cycle FROM fisma_metrics WHERE domain = ? ORDER BY id`,
			domain)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FismaMetric
	for rows.Next() {
		var m FismaMetric
		if err := rows.Scan(&m.ID, &m.Domain, &m.Question, &m.ReviewCycle); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// getMetricsByControl returns all FISMA metrics that reference a given NIST 800-53 control.
func (r *relationalDB) getMetricsByControl(ctx context.Context, controlID string) ([]FismaMetric, error) {
	normalized := nist_800_53.NormalizeID(controlID)
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT m.id, m.domain, m.question, m.review_cycle
		 FROM fisma_metrics m
		 JOIN fisma_criteria c ON c.metric_id = m.id
		 WHERE c.control_id = ?
		 ORDER BY m.id`, normalized)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FismaMetric
	for rows.Next() {
		var m FismaMetric
		if err := rows.Scan(&m.ID, &m.Domain, &m.Question, &m.ReviewCycle); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func seedBaselines(db *sql.DB, baselines map[string][]string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin baselines tx: %w", err)
	}
	defer tx.Rollback()

	for controlID, bls := range baselines {
		for _, bl := range bls {
			if _, err := tx.Exec(
				`INSERT OR IGNORE INTO control_baselines (control_id, baseline) VALUES (?, ?)`,
				controlID, bl,
			); err != nil {
				return fmt.Errorf("insert baseline %s/%s: %w", controlID, bl, err)
			}
		}
	}
	return tx.Commit()
}

func (r *relationalDB) controlBaselines(ctx context.Context, controlID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT baseline FROM control_baselines WHERE control_id = ? ORDER BY baseline`, controlID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var bl string
		if err := rows.Scan(&bl); err != nil {
			return nil, err
		}
		out = append(out, bl)
	}
	return out, rows.Err()
}

func (r *relationalDB) getControl(ctx context.Context, id string) (*nist_800_53.Control, error) {
	normalized := nist_800_53.NormalizeID(id)
	row := r.db.QueryRowContext(ctx,
		`SELECT id, family_id, title, statement, discussion, is_enhancement, COALESCE(parent_id,'')
		 FROM controls WHERE id = ?`, normalized)

	c, err := scanControl(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("control %q not found", id)
	}
	if err != nil {
		return nil, err
	}
	c.Baselines, err = r.controlBaselines(ctx, c.ID)
	return c, err
}

func (r *relationalDB) getBaseline(ctx context.Context, baseline string) ([]nist_800_53.Control, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.family_id, c.title, c.statement, c.discussion, c.is_enhancement, COALESCE(c.parent_id,'')
		 FROM controls c
		 JOIN control_baselines b ON b.control_id = c.id
		 WHERE b.baseline = ?
		 ORDER BY c.id`, baseline)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	controls, err := scanControls(rows)
	if err != nil {
		return nil, err
	}

	// Populate baselines for each returned control.
	for i := range controls {
		controls[i].Baselines, err = r.controlBaselines(ctx, controls[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return controls, nil
}

func (r *relationalDB) listFamilies(ctx context.Context) ([]nist_800_53.Family, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, title FROM families ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []nist_800_53.Family
	for rows.Next() {
		var f nist_800_53.Family
		if err := rows.Scan(&f.ID, &f.Title); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *relationalDB) getFamily(ctx context.Context, familyID string) ([]nist_800_53.Control, error) {
	id := nist_800_53.NormalizeID(familyID)
	// NormalizeID on a plain family ID like "AC" returns "AC" unchanged.
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, family_id, title, statement, discussion, is_enhancement, COALESCE(parent_id,'')
		 FROM controls WHERE family_id = ? AND is_enhancement = 0 ORDER BY id`,
		id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanControls(rows)
}

func (r *relationalDB) search(ctx context.Context, query string, limit int, source string) ([]SearchResult, error) {
	ftsQuery := sanitizeFTS(query)
	if ftsQuery == "" {
		return nil, nil
	}

	var out []SearchResult

	if source == "" || source == "nist_800_53" {
		res, err := r.searchControlsFTS(ctx, ftsQuery, limit)
		if err != nil {
			return nil, err
		}
		out = append(out, res...)
	}

	if source == "" || source == "fisma_fy2025" {
		res, err := r.searchFismaFTS(ctx, ftsQuery, limit)
		if err != nil {
			return nil, err
		}
		out = append(out, res...)
	}

	if source == "" || source == "nist_csf_v2" {
		res, err := r.searchCSFFTS(ctx, ftsQuery, limit)
		if err != nil {
			return nil, err
		}
		out = append(out, res...)
	}

	if source == "" || source == "fedramp_20x" {
		res, err := r.searchKSIFTS(ctx, ftsQuery, limit)
		if err != nil {
			return nil, err
		}
		out = append(out, res...)
		res, err = r.searchFedRAMPReqFTS(ctx, ftsQuery, limit)
		if err != nil {
			return nil, err
		}
		out = append(out, res...)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Relevance > out[j].Relevance })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *relationalDB) searchControlsFTS(ctx context.Context, ftsQuery string, limit int) ([]SearchResult, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.title, c.statement, -bm25(controls_fts) AS score
		 FROM controls_fts
		 JOIN controls c ON c.id = controls_fts.id
		 WHERE controls_fts MATCH ?
		 ORDER BY bm25(controls_fts)
		 LIMIT ?`,
		ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("fts search controls: %w", err)
	}
	defer rows.Close()

	type rawRow struct {
		id, title, statement string
		score                float64
	}
	var raw []rawRow
	for rows.Next() {
		var rr rawRow
		if err := rows.Scan(&rr.id, &rr.title, &rr.statement, &rr.score); err != nil {
			return nil, err
		}
		raw = append(raw, rr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}

	maxScore := raw[0].score
	results := make([]SearchResult, len(raw))
	for i, rr := range raw {
		rel := float32(1.0)
		if maxScore > 0 {
			rel = float32(rr.score / maxScore)
		}
		results[i] = SearchResult{
			Source:    "nist_800_53",
			ID:        rr.id,
			Title:     rr.id + " " + rr.title,
			Body:      rr.statement,
			Relevance: rel,
		}
	}
	return results, nil
}

func (r *relationalDB) searchFismaFTS(ctx context.Context, ftsQuery string, limit int) ([]SearchResult, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT m.id, m.domain, m.question, -bm25(fisma_metrics_fts) AS score
		 FROM fisma_metrics_fts
		 JOIN fisma_metrics m ON m.id = fisma_metrics_fts.id
		 WHERE fisma_metrics_fts MATCH ?
		 ORDER BY bm25(fisma_metrics_fts)
		 LIMIT ?`,
		ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("fts search fisma metrics: %w", err)
	}
	defer rows.Close()

	type rawRow struct {
		id       int
		domain   string
		question string
		score    float64
	}
	var raw []rawRow
	for rows.Next() {
		var rr rawRow
		if err := rows.Scan(&rr.id, &rr.domain, &rr.question, &rr.score); err != nil {
			return nil, err
		}
		raw = append(raw, rr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}

	maxScore := raw[0].score
	results := make([]SearchResult, len(raw))
	for i, rr := range raw {
		rel := float32(1.0)
		if maxScore > 0 {
			rel = float32(rr.score / maxScore)
		}
		results[i] = SearchResult{
			Source:    "fisma_fy2025",
			ID:        fmt.Sprintf("%d", rr.id),
			Title:     rr.domain,
			Body:      rr.question,
			Relevance: rel,
		}
	}
	return results, nil
}

func seedCSF(db *sql.DB, fns []nist_csf.Function, cats []nist_csf.Category, subs []nist_csf.Subcategory) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin csf tx: %w", err)
	}
	defer tx.Rollback()

	for _, f := range fns {
		if _, err := tx.Exec(`INSERT INTO csf_functions (id, title, text) VALUES (?, ?, ?)`, f.ID, f.Title, f.Text); err != nil {
			return fmt.Errorf("insert csf function %s: %w", f.ID, err)
		}
	}

	for _, c := range cats {
		if _, err := tx.Exec(`INSERT INTO csf_categories (id, function_id, title, text) VALUES (?, ?, ?, ?)`, c.ID, c.FunctionID, c.Title, c.Text); err != nil {
			return fmt.Errorf("insert csf category %s: %w", c.ID, err)
		}
	}

	for _, s := range subs {
		if _, err := tx.Exec(`INSERT INTO csf_subcategories (id, category_id, function_id, text) VALUES (?, ?, ?, ?)`, s.ID, s.CategoryID, s.FunctionID, s.Text); err != nil {
			return fmt.Errorf("insert csf subcategory %s: %w", s.ID, err)
		}
		if _, err := tx.Exec(`INSERT INTO csf_subcategories_fts (id, category_id, function_id, text) VALUES (?, ?, ?, ?)`, s.ID, s.CategoryID, s.FunctionID, s.Text); err != nil {
			return fmt.Errorf("insert csf subcategory fts %s: %w", s.ID, err)
		}
		for _, ex := range s.Examples {
			if ex == "" {
				continue
			}
			if _, err := tx.Exec(`INSERT INTO csf_examples (subcategory_id, text) VALUES (?, ?)`, s.ID, ex); err != nil {
				return fmt.Errorf("insert csf example %s: %w", s.ID, err)
			}
		}
	}

	return tx.Commit()
}

func (r *relationalDB) getCSFSubcategory(ctx context.Context, id string) (*nist_csf.Subcategory, error) {
	id = strings.ToUpper(id)
	row := r.db.QueryRowContext(ctx,
		`SELECT id, category_id, function_id, text FROM csf_subcategories WHERE id = ?`, id)
	var s nist_csf.Subcategory
	if err := row.Scan(&s.ID, &s.CategoryID, &s.FunctionID, &s.Text); err == sql.ErrNoRows {
		return nil, fmt.Errorf("csf subcategory %q not found", id)
	} else if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `SELECT text FROM csf_examples WHERE subcategory_id = ? ORDER BY id`, s.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ex string
		if err := rows.Scan(&ex); err != nil {
			return nil, err
		}
		s.Examples = append(s.Examples, ex)
	}
	return &s, rows.Err()
}

func (r *relationalDB) listCSFCategories(ctx context.Context, functionID string) ([]nist_csf.Category, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if functionID == "" {
		rows, err = r.db.QueryContext(ctx, `SELECT id, function_id, title, text FROM csf_categories ORDER BY id`)
	} else {
		rows, err = r.db.QueryContext(ctx, `SELECT id, function_id, title, text FROM csf_categories WHERE function_id = ? ORDER BY id`, strings.ToUpper(functionID))
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []nist_csf.Category
	for rows.Next() {
		var c nist_csf.Category
		if err := rows.Scan(&c.ID, &c.FunctionID, &c.Title, &c.Text); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *relationalDB) listCSFFunctions(ctx context.Context) ([]nist_csf.Function, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, title, text FROM csf_functions ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []nist_csf.Function
	for rows.Next() {
		var f nist_csf.Function
		if err := rows.Scan(&f.ID, &f.Title, &f.Text); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *relationalDB) searchCSFFTS(ctx context.Context, ftsQuery string, limit int) ([]SearchResult, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT s.id, s.category_id, s.text, -bm25(csf_subcategories_fts) AS score
		 FROM csf_subcategories_fts
		 JOIN csf_subcategories s ON s.id = csf_subcategories_fts.id
		 WHERE csf_subcategories_fts MATCH ?
		 ORDER BY bm25(csf_subcategories_fts)
		 LIMIT ?`,
		ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("fts search csf subcategories: %w", err)
	}
	defer rows.Close()

	type rawRow struct {
		id, catID, text string
		score           float64
	}
	var raw []rawRow
	for rows.Next() {
		var rr rawRow
		if err := rows.Scan(&rr.id, &rr.catID, &rr.text, &rr.score); err != nil {
			return nil, err
		}
		raw = append(raw, rr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}

	maxScore := raw[0].score
	results := make([]SearchResult, len(raw))
	for i, rr := range raw {
		rel := float32(1.0)
		if maxScore > 0 {
			rel = float32(rr.score / maxScore)
		}
		results[i] = SearchResult{
			Source:    "nist_csf_v2",
			ID:        rr.id,
			Title:     rr.id,
			Body:      rr.text,
			Relevance: rel,
		}
	}
	return results, nil
}

func seedFedRAMP(db *sql.DB, frmr *fedramp.Catalog) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin fedramp tx: %w", err)
	}
	defer tx.Rollback()

	for _, t := range frmr.Terms {
		if _, err := tx.Exec(`INSERT INTO fedramp_terms (id, term, definition, note) VALUES (?, ?, ?, ?)`,
			t.ID, t.Term, t.Definition, t.Note); err != nil {
			return fmt.Errorf("insert fedramp term %s: %w", t.ID, err)
		}
		if _, err := tx.Exec(`INSERT INTO fedramp_terms_fts (id, term, definition) VALUES (?, ?, ?)`,
			t.ID, t.Term, t.Definition); err != nil {
			return fmt.Errorf("insert fedramp term fts %s: %w", t.ID, err)
		}
	}

	for _, theme := range frmr.KSIThemes {
		if _, err := tx.Exec(`INSERT INTO ksi_themes (id, short_name, name, theme) VALUES (?, ?, ?, ?)`,
			theme.ID, theme.ShortName, theme.Name, theme.Theme); err != nil {
			return fmt.Errorf("insert ksi theme %s: %w", theme.ID, err)
		}
		for _, ind := range theme.Indicators {
			if _, err := tx.Exec(`INSERT INTO ksi_indicators (id, theme_id, name, statement) VALUES (?, ?, ?, ?)`,
				ind.ID, ind.ThemeID, ind.Name, ind.Statement); err != nil {
				return fmt.Errorf("insert ksi indicator %s: %w", ind.ID, err)
			}
			if _, err := tx.Exec(`INSERT INTO ksi_indicators_fts (id, theme_id, name, statement) VALUES (?, ?, ?, ?)`,
				ind.ID, ind.ThemeID, ind.Name, ind.Statement); err != nil {
				return fmt.Errorf("insert ksi indicator fts %s: %w", ind.ID, err)
			}
			for _, ctrl := range ind.Controls {
				if _, err := tx.Exec(`INSERT OR IGNORE INTO ksi_controls (indicator_id, control_id) VALUES (?, ?)`,
					ind.ID, ctrl); err != nil {
					return fmt.Errorf("insert ksi control %s/%s: %w", ind.ID, ctrl, err)
				}
			}
		}
	}

	for _, rc := range frmr.Requirements {
		for _, req := range rc.Requirements {
			if _, err := tx.Exec(`INSERT INTO fedramp_requirements (id, category, name, statement, keyword, version) VALUES (?, ?, ?, ?, ?, ?)`,
				req.ID, req.Category, req.Name, req.Statement, req.Keyword, req.Version); err != nil {
				return fmt.Errorf("insert fedramp requirement %s: %w", req.ID, err)
			}
			if _, err := tx.Exec(`INSERT INTO fedramp_requirements_fts (id, category, name, statement) VALUES (?, ?, ?, ?)`,
				req.ID, req.Category, req.Name, req.Statement); err != nil {
				return fmt.Errorf("insert fedramp requirement fts %s: %w", req.ID, err)
			}
		}
	}

	return tx.Commit()
}

func (r *relationalDB) getKSI(ctx context.Context, id string) (*fedramp.KSIIndicator, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT k.id, k.theme_id, k.name, k.statement FROM ksi_indicators k WHERE k.id = ?`,
		strings.ToUpper(id))
	var ind fedramp.KSIIndicator
	if err := row.Scan(&ind.ID, &ind.ThemeID, &ind.Name, &ind.Statement); err == sql.ErrNoRows {
		return nil, fmt.Errorf("KSI indicator %q not found", id)
	} else if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT control_id FROM ksi_controls WHERE indicator_id = ? ORDER BY control_id`, ind.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		ind.Controls = append(ind.Controls, c)
	}
	return &ind, rows.Err()
}

func (r *relationalDB) listKSIThemes(ctx context.Context) ([]fedramp.KSITheme, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, short_name, name, theme FROM ksi_themes ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var themes []fedramp.KSITheme
	for rows.Next() {
		var t fedramp.KSITheme
		if err := rows.Scan(&t.ID, &t.ShortName, &t.Name, &t.Theme); err != nil {
			return nil, err
		}
		themes = append(themes, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Populate indicators for each theme.
	for i, theme := range themes {
		irows, err := r.db.QueryContext(ctx,
			`SELECT id, theme_id, name, statement FROM ksi_indicators WHERE theme_id = ? ORDER BY id`, theme.ID)
		if err != nil {
			return nil, err
		}
		defer irows.Close()
		for irows.Next() {
			var ind fedramp.KSIIndicator
			if err := irows.Scan(&ind.ID, &ind.ThemeID, &ind.Name, &ind.Statement); err != nil {
				return nil, err
			}
			themes[i].Indicators = append(themes[i].Indicators, ind)
		}
		if err := irows.Err(); err != nil {
			return nil, err
		}
	}
	return themes, nil
}

func (r *relationalDB) getKSIsByControl(ctx context.Context, controlID string) ([]fedramp.KSIIndicator, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT k.id, k.theme_id, k.name, k.statement
		 FROM ksi_indicators k
		 JOIN ksi_controls c ON c.indicator_id = k.id
		 WHERE c.control_id = ?
		 ORDER BY k.id`, strings.ToUpper(controlID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []fedramp.KSIIndicator
	for rows.Next() {
		var ind fedramp.KSIIndicator
		if err := rows.Scan(&ind.ID, &ind.ThemeID, &ind.Name, &ind.Statement); err != nil {
			return nil, err
		}
		out = append(out, ind)
	}
	return out, rows.Err()
}

func (r *relationalDB) listFedRAMPRequirements(ctx context.Context, category, version string) ([]fedramp.Requirement, error) {
	q := `SELECT id, category, name, statement, keyword, version FROM fedramp_requirements WHERE 1=1`
	var args []any
	if category != "" {
		q += ` AND category = ?`
		args = append(args, strings.ToUpper(category))
	}
	if version != "" {
		q += ` AND (version = ? OR version = 'both')`
		args = append(args, strings.ToLower(version))
	}
	q += ` ORDER BY category, id`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []fedramp.Requirement
	for rows.Next() {
		var req fedramp.Requirement
		if err := rows.Scan(&req.ID, &req.Category, &req.Name, &req.Statement, &req.Keyword, &req.Version); err != nil {
			return nil, err
		}
		out = append(out, req)
	}
	return out, rows.Err()
}

func (r *relationalDB) getFedRAMPTerm(ctx context.Context, id string) (*fedramp.Term, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, term, definition, note FROM fedramp_terms WHERE id = ?`, strings.ToUpper(id))
	var t fedramp.Term
	if err := row.Scan(&t.ID, &t.Term, &t.Definition, &t.Note); err == sql.ErrNoRows {
		return nil, fmt.Errorf("FedRAMP term %q not found", id)
	} else if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *relationalDB) searchKSIFTS(ctx context.Context, ftsQuery string, limit int) ([]SearchResult, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT k.id, k.theme_id, k.name, k.statement, -bm25(ksi_indicators_fts) AS score
		 FROM ksi_indicators_fts
		 JOIN ksi_indicators k ON k.id = ksi_indicators_fts.id
		 WHERE ksi_indicators_fts MATCH ?
		 ORDER BY bm25(ksi_indicators_fts)
		 LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("fts search ksi: %w", err)
	}
	defer rows.Close()

	type rawRow struct {
		id, themeID, name, statement string
		score                        float64
	}
	var raw []rawRow
	for rows.Next() {
		var rr rawRow
		if err := rows.Scan(&rr.id, &rr.themeID, &rr.name, &rr.statement, &rr.score); err != nil {
			return nil, err
		}
		raw = append(raw, rr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	maxScore := raw[0].score
	results := make([]SearchResult, len(raw))
	for i, rr := range raw {
		rel := float32(1.0)
		if maxScore > 0 {
			rel = float32(rr.score / maxScore)
		}
		results[i] = SearchResult{
			Source:    "fedramp_20x",
			ID:        rr.id,
			Title:     rr.name,
			Body:      rr.statement,
			Relevance: rel,
		}
	}
	return results, nil
}

func (r *relationalDB) searchFedRAMPReqFTS(ctx context.Context, ftsQuery string, limit int) ([]SearchResult, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT r.id, r.category, r.name, r.statement, -bm25(fedramp_requirements_fts) AS score
		 FROM fedramp_requirements_fts
		 JOIN fedramp_requirements r ON r.id = fedramp_requirements_fts.id
		 WHERE fedramp_requirements_fts MATCH ?
		 ORDER BY bm25(fedramp_requirements_fts)
		 LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("fts search fedramp requirements: %w", err)
	}
	defer rows.Close()

	type rawRow struct {
		id, category, name, statement string
		score                         float64
	}
	var raw []rawRow
	for rows.Next() {
		var rr rawRow
		if err := rows.Scan(&rr.id, &rr.category, &rr.name, &rr.statement, &rr.score); err != nil {
			return nil, err
		}
		raw = append(raw, rr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	maxScore := raw[0].score
	results := make([]SearchResult, len(raw))
	for i, rr := range raw {
		rel := float32(1.0)
		if maxScore > 0 {
			rel = float32(rr.score / maxScore)
		}
		results[i] = SearchResult{
			Source:    "fedramp_20x",
			ID:        rr.id,
			Title:     rr.category + " — " + rr.name,
			Body:      rr.statement,
			Relevance: rel,
		}
	}
	return results, nil
}

// sanitizeFTS strips characters that have special meaning in FTS5 query syntax,
// preventing parse errors on arbitrary user input.
func sanitizeFTS(q string) string {
	var b strings.Builder
	for _, r := range q {
		switch r {
		case '"', '(', ')', '*', '^', '-', ':':
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func scanControl(row *sql.Row) (*nist_800_53.Control, error) {
	var c nist_800_53.Control
	var enhancement int
	if err := row.Scan(&c.ID, &c.FamilyID, &c.Title, &c.Statement, &c.Discussion, &enhancement, &c.ParentID); err != nil {
		return nil, err
	}
	c.IsEnhancement = enhancement == 1
	return &c, nil
}

func scanControls(rows *sql.Rows) ([]nist_800_53.Control, error) {
	var out []nist_800_53.Control
	for rows.Next() {
		var c nist_800_53.Control
		var enhancement int
		if err := rows.Scan(&c.ID, &c.FamilyID, &c.Title, &c.Statement, &c.Discussion, &enhancement, &c.ParentID); err != nil {
			return nil, err
		}
		c.IsEnhancement = enhancement == 1
		out = append(out, c)
	}
	return out, rows.Err()
}
