package rel_store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/fedramp"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/fisma"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_800_53"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_csf"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/vec_store"
)

// SearchResult is a source-agnostic search hit returned by Search.
// Source identifies which document corpus the hit came from ("nist_800_53", "fisma_fy2025", …).
// Use the source-specific Get methods to retrieve the full record.
type SearchResult struct {
	Source    string  `json:"source"`
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	Relevance float32 `json:"relevance"`
}

// Store provides deterministic and semantic access to NIST SP 800-53 controls.
type Store struct {
	rel *relationalDB
	vec *vec_store.VectorDB // nil when no embedding provider is configured
}

// Config holds embedding provider settings. Leave EmbeddingProvider empty to
// disable vector search (relational lookups remain fully functional).
type Config struct {
	EmbeddingProvider string // "openai" | "ollama" | "" (disabled)
	EmbeddingModel    string
	OpenAIKey         string
	OllamaBaseURL     string
}

// New loads the embedded NIST catalog, populates the in-memory relational DB,
// and optionally builds the chromem-go vector index.
func New(ctx context.Context, cfg Config) (*Store, error) {
	// Infer provider and model from the pre-built index when not explicitly set.
	// Each binary variant embeds exactly one index, so the meta is authoritative.
	if _, meta, ok := vec_store.PrebuiltVector(); ok {
		if cfg.EmbeddingProvider == "" {
			cfg.EmbeddingProvider = meta.Provider
		}
		if cfg.EmbeddingModel == "" && meta.Provider == cfg.EmbeddingProvider {
			cfg.EmbeddingModel = meta.Model
		}
	}

	families, controls, err := nist_800_53.Load()
	if err != nil {
		return nil, err
	}

	baselines, err := nist_800_53.LoadBaselines()
	if err != nil {
		return nil, err
	}

	metrics, err := fisma.Load()
	if err != nil {
		return nil, err
	}

	csfFunctions, csfCategories, csfSubcategories, err := nist_csf.Load()
	if err != nil {
		return nil, err
	}

	csfCrosswalk, err := nist_csf.LoadCrosswalk()
	if err != nil {
		return nil, err
	}

	frmr, err := fedramp.Load()
	if err != nil {
		return nil, err
	}

	rel, err := newRelationalDB(families, controls, baselines, metrics, csfFunctions, csfCategories, csfSubcategories, csfCrosswalk, frmr)
	if err != nil {
		return nil, err
	}

	var vec *vec_store.VectorDB
	if cfg.EmbeddingProvider != "" {
		vec, err = vec_store.NewVectorDB(ctx, vec_store.EmbedConfig{
			Provider:  cfg.EmbeddingProvider,
			Model:     cfg.EmbeddingModel,
			OpenAIKey: cfg.OpenAIKey,
			OllamaURL: cfg.OllamaBaseURL,
		}, controls, metrics, csfSubcategories, frmr)
		if err != nil {
			return nil, err
		}
	}

	return &Store{rel: rel, vec: vec}, nil
}

// Close releases underlying resources.
func (s *Store) Close() error {
	return s.rel.close()
}

// GetControl returns a single control by its canonical ID (e.g., "AC-1" or "ac-1").
func (s *Store) GetControl(ctx context.Context, id string) (*nist_800_53.Control, error) {
	return s.rel.getControl(ctx, id)
}

// ListFamilies returns all control families.
func (s *Store) ListFamilies(ctx context.Context) ([]nist_800_53.Family, error) {
	return s.rel.listFamilies(ctx)
}

// GetFamily returns all controls (base controls only, not enhancements) for a family.
func (s *Store) GetFamily(ctx context.Context, familyID string) ([]nist_800_53.Control, error) {
	return s.rel.getFamily(ctx, familyID)
}

// Search performs semantic vector search when an embedding provider is configured,
// or FTS5 full-text search otherwise. Pass a non-empty source to restrict results
// to a single corpus ("nist_800_53" or "fisma_fy2025"); pass "" to search all.
func (s *Store) Search(ctx context.Context, query string, limit int, source string) ([]SearchResult, error) {
	if s.vec != nil {
		hits, err := s.vec.Query(ctx, query, limit, source)
		if err != nil {
			return nil, err
		}
		return s.resolveHits(ctx, hits)
	}
	return s.rel.search(ctx, query, limit, source)
}

func (s *Store) resolveHits(ctx context.Context, hits []vec_store.RawHit) ([]SearchResult, error) {
	results := make([]SearchResult, 0, len(hits))
	for _, h := range hits {
		switch h.Source {
		case "nist_800_53":
			c, err := s.rel.getControl(ctx, h.ID)
			if err != nil {
				continue
			}
			results = append(results, SearchResult{
				Source:    "nist_800_53",
				ID:        c.ID,
				Title:     c.ID + " " + c.Title,
				Body:      c.Statement,
				Relevance: h.Similarity,
			})
		case "fisma_fy2025":
			id, _ := strconv.Atoi(h.ID)
			m, err := s.rel.getFismaMetric(ctx, id)
			if err != nil {
				continue
			}
			results = append(results, SearchResult{
				Source:    "fisma_fy2025",
				ID:        h.ID,
				Title:     m.Domain,
				Body:      m.Question,
				Relevance: h.Similarity,
			})
		case "nist_csf_v2":
			sub, err := s.rel.getCSFSubcategory(ctx, h.ID)
			if err != nil {
				continue
			}
			results = append(results, SearchResult{
				Source:    "nist_csf_v2",
				ID:        sub.ID,
				Title:     sub.ID,
				Body:      sub.Text,
				Relevance: h.Similarity,
			})
		case "fedramp_20x":
			switch h.DocType {
			case "ksi":
				ind, err := s.rel.getKSI(ctx, h.ID)
				if err != nil {
					continue
				}
				results = append(results, SearchResult{
					Source:    "fedramp_20x",
					ID:        ind.ID,
					Title:     ind.Name,
					Body:      ind.Statement,
					Relevance: h.Similarity,
				})
			case "term":
				term, err := s.rel.getFedRAMPTerm(ctx, h.ID)
				if err != nil {
					continue
				}
				results = append(results, SearchResult{
					Source:    "fedramp_20x",
					ID:        term.ID,
					Title:     term.Term,
					Body:      term.Definition,
					Relevance: h.Similarity,
				})
			default: // "requirement"
				req, err := s.rel.getFedRAMPRequirement(ctx, h.ID)
				if err != nil {
					continue
				}
				results = append(results, SearchResult{
					Source:    "fedramp_20x",
					ID:        req.ID,
					Title:     req.Name,
					Body:      req.Statement,
					Relevance: h.Similarity,
				})
			}
		}
	}
	return results, nil
}

// GetFismaMetric returns a single FY 2025 IG FISMA metric by its numeric ID,
// including all maturity levels and criteria references.
func (s *Store) GetFismaMetric(ctx context.Context, id int) (*FismaMetric, error) {
	return s.rel.getFismaMetric(ctx, id)
}

// ListFismaMetrics returns all FY 2025 IG FISMA metrics. Pass a non-empty domain
// string to filter to a specific domain (e.g., "Identity Management and Access Control").
func (s *Store) ListFismaMetrics(ctx context.Context, domain string) ([]FismaMetric, error) {
	return s.rel.listFismaMetrics(ctx, domain)
}

// GetMetricsByControl returns all FISMA metrics that reference a given NIST SP
// 800-53 control ID.
func (s *Store) GetMetricsByControl(ctx context.Context, controlID string) ([]FismaMetric, error) {
	return s.rel.getMetricsByControl(ctx, controlID)
}

// GetMetricsByCSFSubcategory returns all FISMA metrics that reference a given
// NIST CSF 2.0 subcategory ID (e.g., "GV.OC-01").
func (s *Store) GetMetricsByCSFSubcategory(ctx context.Context, subcategoryID string) ([]FismaMetric, error) {
	return s.rel.getMetricsByCSFSubcategory(ctx, subcategoryID)
}

// GetBaseline returns all NIST SP 800-53 controls and enhancements included in
// the given SP 800-53B baseline. Accepted values: "low", "moderate", "high", "privacy".
func (s *Store) GetBaseline(ctx context.Context, baseline string) ([]nist_800_53.Control, error) {
	name := nist_800_53.NormalizeBaseline(baseline)
	if name == "" {
		return nil, fmt.Errorf("unknown baseline %q: use low, moderate, high, or privacy", baseline)
	}
	return s.rel.getBaseline(ctx, name)
}

// GetCSFSubcategory returns a single NIST CSF v2.0 subcategory by its identifier
// (e.g., "GV.OC-01"), including all implementation examples.
func (s *Store) GetCSFSubcategory(ctx context.Context, id string) (*nist_csf.Subcategory, error) {
	return s.rel.getCSFSubcategory(ctx, id)
}

// ListCSFCategories returns all NIST CSF v2.0 categories. Pass a non-empty
// functionID (e.g., "GV") to filter to a single function.
func (s *Store) ListCSFCategories(ctx context.Context, functionID string) ([]nist_csf.Category, error) {
	return s.rel.listCSFCategories(ctx, functionID)
}

// ListCSFFunctions returns all six NIST CSF v2.0 functions.
func (s *Store) ListCSFFunctions(ctx context.Context) ([]nist_csf.Function, error) {
	return s.rel.listCSFFunctions(ctx)
}

// GetCSFSubcategoriesByControl returns all CSF 2.0 subcategories that map to a
// given NIST SP 800-53 control ID per the official crosswalk.
func (s *Store) GetCSFSubcategoriesByControl(ctx context.Context, controlID string) ([]nist_csf.Subcategory, error) {
	normalized := nist_800_53.NormalizeID(controlID)
	return s.rel.getCSFSubcategoriesByControl(ctx, normalized)
}

// ListKSIThemes returns all FedRAMP 20x KSI themes with their indicators.
func (s *Store) ListKSIThemes(ctx context.Context) ([]fedramp.KSITheme, error) {
	return s.rel.listKSIThemes(ctx)
}

// GetKSI returns a single FedRAMP 20x KSI indicator by its ID (e.g., "KSI-IAM-MFA").
func (s *Store) GetKSI(ctx context.Context, id string) (*fedramp.KSIIndicator, error) {
	return s.rel.getKSI(ctx, id)
}

// GetKSIsByControl returns all FedRAMP 20x KSI indicators that reference a given
// NIST SP 800-53 control ID.
func (s *Store) GetKSIsByControl(ctx context.Context, controlID string) ([]fedramp.KSIIndicator, error) {
	normalized := nist_800_53.NormalizeID(controlID)
	return s.rel.getKSIsByControl(ctx, normalized)
}

// ListFedRAMPRequirements returns FedRAMP process requirements. Optionally filter
// by category (e.g., "VDR") and/or version ("rev5", "20x", "both").
func (s *Store) ListFedRAMPRequirements(ctx context.Context, category, version string) ([]fedramp.Requirement, error) {
	return s.rel.listFedRAMPRequirements(ctx, category, version)
}

// GetFedRAMPTerm returns a single FedRAMP glossary term by its ID (e.g., "FRD-ACV").
func (s *Store) GetFedRAMPTerm(ctx context.Context, id string) (*fedramp.Term, error) {
	return s.rel.getFedRAMPTerm(ctx, id)
}

// GetFedRAMPRequirement returns a single FedRAMP process requirement by its ID (e.g., "VDR-BST-SCA").
func (s *Store) GetFedRAMPRequirement(ctx context.Context, id string) (*fedramp.Requirement, error) {
	return s.rel.getFedRAMPRequirement(ctx, id)
}
