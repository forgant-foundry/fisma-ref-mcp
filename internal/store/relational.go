package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist"
	_ "modernc.org/sqlite"
)

type relationalDB struct {
	db *sql.DB
}

func newRelationalDB(families []nist.Family, controls []nist.Control) (*relationalDB, error) {
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

		CREATE VIRTUAL TABLE controls_fts USING fts5(
			id       UNINDEXED,
			title,
			statement,
			discussion,
			tokenize = 'unicode61'
		);
	`)
	return err
}

func seed(db *sql.DB, families []nist.Family, controls []nist.Control) error {
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

func (r *relationalDB) getControl(ctx context.Context, id string) (*nist.Control, error) {
	normalized := nist.NormalizeID(id)
	row := r.db.QueryRowContext(ctx,
		`SELECT id, family_id, title, statement, discussion, is_enhancement, COALESCE(parent_id,'')
		 FROM controls WHERE id = ?`, normalized)

	c, err := scanControl(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("control %q not found", id)
	}
	return c, err
}

func (r *relationalDB) listFamilies(ctx context.Context) ([]nist.Family, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, title FROM families ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []nist.Family
	for rows.Next() {
		var f nist.Family
		if err := rows.Scan(&f.ID, &f.Title); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *relationalDB) getFamily(ctx context.Context, familyID string) ([]nist.Control, error) {
	id := nist.NormalizeID(familyID)
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

func (r *relationalDB) search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	ftsQuery := sanitizeFTS(query)
	if ftsQuery == "" {
		return nil, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.family_id, c.title, c.statement, c.discussion,
		        c.is_enhancement, COALESCE(c.parent_id,''),
		        -bm25(controls_fts) AS score
		 FROM controls_fts
		 JOIN controls c ON c.id = controls_fts.id
		 WHERE controls_fts MATCH ?
		 ORDER BY bm25(controls_fts)
		 LIMIT ?`,
		ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer rows.Close()

	type row struct {
		control nist.Control
		score   float64
	}
	var raw []row
	for rows.Next() {
		var c nist.Control
		var enhancement int
		var score float64
		if err := rows.Scan(&c.ID, &c.FamilyID, &c.Title, &c.Statement, &c.Discussion, &enhancement, &c.ParentID, &score); err != nil {
			return nil, err
		}
		c.IsEnhancement = enhancement == 1
		raw = append(raw, row{c, score})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(raw) == 0 {
		return nil, nil
	}

	// Normalize scores relative to the top result so Relevance is in (0, 1].
	maxScore := raw[0].score
	results := make([]SearchResult, len(raw))
	for i, r := range raw {
		rel := float32(1.0)
		if maxScore > 0 {
			rel = float32(r.score / maxScore)
		}
		results[i] = SearchResult{Control: r.control, Relevance: rel}
	}
	return results, nil
}

// sanitizeFTS strips characters that have special meaning in FTS5 query syntax,
// preventing parse errors on arbitrary user input.
func sanitizeFTS(q string) string {
	var b strings.Builder
	for _, r := range q {
		switch r {
		case '"', '(', ')', '*', '^':
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func scanControl(row *sql.Row) (*nist.Control, error) {
	var c nist.Control
	var enhancement int
	if err := row.Scan(&c.ID, &c.FamilyID, &c.Title, &c.Statement, &c.Discussion, &enhancement, &c.ParentID); err != nil {
		return nil, err
	}
	c.IsEnhancement = enhancement == 1
	return &c, nil
}

func scanControls(rows *sql.Rows) ([]nist.Control, error) {
	var out []nist.Control
	for rows.Next() {
		var c nist.Control
		var enhancement int
		if err := rows.Scan(&c.ID, &c.FamilyID, &c.Title, &c.Statement, &c.Discussion, &enhancement, &c.ParentID); err != nil {
			return nil, err
		}
		c.IsEnhancement = enhancement == 1
		out = append(out, c)
	}
	return out, rows.Err()
}
