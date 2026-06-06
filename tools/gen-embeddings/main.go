//go:build ignore

// gen-embeddings builds the chromem-go vector index from the embedded NIST
// SP 800-53 Rev 5 catalog and writes the serialised result into
// internal/nist/data/ so it can be compiled into the binary via //go:embed.
//
// Usage:
//
//	OPENAI_API_KEY=sk-... go run ./tools/gen-embeddings/main.go --provider openai
//	OLLAMA_URL=http://localhost:11434 go run ./tools/gen-embeddings/main.go --provider ollama --model nomic-embed-text
//
// Or via go generate from the repo root:
//
//	EMBEDDING_PROVIDER=openai OPENAI_API_KEY=sk-... go generate ./internal/nist
//
// After running, rebuild the binary to embed the new index:
//
//	go build .
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/fedramp"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/fisma"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_800_53"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_csf"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/vec_store"
	"github.com/philippgille/chromem-go"
)

func main() {
	provider := flag.String("provider", envOr("EMBEDDING_PROVIDER", ""), `Embedding provider: "openai" or "ollama" (required)`)
	model := flag.String("model", envOr("EMBEDDING_MODEL", ""), "Model name (uses provider default when omitted)")
	ollamaURL := flag.String("ollama-url", envOr("OLLAMA_URL", "http://localhost:11434"), "Ollama base URL")
	outputDir := flag.String("output-dir", "", "Output directory for chromem.db and chromem-meta.json (default: internal/nist/data relative to repo root)")
	flag.Parse()

	if *provider == "" {
		flag.Usage()
		log.Fatal("--provider is required")
	}

	embFn, effectiveModel, err := makeEmbeddingFunc(*provider, *model, *ollamaURL)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	_, controls, err := nist_800_53.Load()
	if err != nil {
		log.Fatalf("load NIST catalog: %v", err)
	}

	metrics, err := fisma.Load()
	if err != nil {
		log.Fatalf("load FISMA metrics: %v", err)
	}

	_, _, subcategories, err := nist_csf.Load()
	if err != nil {
		log.Fatalf("load CSF subcategories: %v", err)
	}

	frmr, err := fedramp.Load()
	if err != nil {
		log.Fatalf("load FedRAMP catalog: %v", err)
	}

	log.Printf("indexing %d controls + %d FISMA metrics + %d CSF subcategories + FedRAMP with %s/%s ...", len(controls), len(metrics), len(subcategories), *provider, effectiveModel)

	db := chromem.NewDB()

	// NIST SP 800-53 controls collection
	controlCol, err := db.GetOrCreateCollection("controls", nil, embFn)
	if err != nil {
		log.Fatalf("create controls collection: %v", err)
	}

	controlDocs := make([]chromem.Document, 0, len(controls))
	for _, c := range controls {
		if content := vec_store.BuildControlDocument(c); content != "" {
			controlDocs = append(controlDocs, chromem.Document{
				ID:      strings.ToUpper(c.ID),
				Content: content,
				Metadata: map[string]string{
					"family":         c.FamilyID,
					"is_enhancement": fmt.Sprintf("%v", c.IsEnhancement),
				},
			})
		}
	}
	if err := controlCol.AddDocuments(ctx, controlDocs, runtime.NumCPU()); err != nil {
		log.Fatalf("embed controls: %v", err)
	}
	log.Printf("embedded %d controls", len(controlDocs))

	// FY 2025 IG FISMA metrics collection
	fismaCol, err := db.GetOrCreateCollection("fisma_metrics", nil, embFn)
	if err != nil {
		log.Fatalf("create fisma_metrics collection: %v", err)
	}

	fismaDocs := make([]chromem.Document, 0, len(metrics))
	for _, m := range metrics {
		if content := vec_store.BuildMetricDocument(m); content != "" {
			fismaDocs = append(fismaDocs, chromem.Document{
				ID:      fmt.Sprintf("%d", m.ID),
				Content: content,
				Metadata: map[string]string{
					"domain":       m.Domain,
					"review_cycle": m.ReviewCycle,
				},
			})
		}
	}
	if err := fismaCol.AddDocuments(ctx, fismaDocs, runtime.NumCPU()); err != nil {
		log.Fatalf("embed fisma metrics: %v", err)
	}
	log.Printf("embedded %d FISMA metrics", len(fismaDocs))

	// NIST CSF 2.0 subcategories collection
	csfCol, err := db.GetOrCreateCollection("csf_v2", nil, embFn)
	if err != nil {
		log.Fatalf("create csf_v2 collection: %v", err)
	}

	csfDocs := make([]chromem.Document, 0, len(subcategories))
	for _, s := range subcategories {
		if content := vec_store.BuildSubcategoryDocument(s); content != "" {
			csfDocs = append(csfDocs, chromem.Document{
				ID:      s.ID,
				Content: content,
				Metadata: map[string]string{
					"category_id": s.CategoryID,
					"function_id": s.FunctionID,
				},
			})
		}
	}
	if err := csfCol.AddDocuments(ctx, csfDocs, runtime.NumCPU()); err != nil {
		log.Fatalf("embed csf subcategories: %v", err)
	}
	log.Printf("embedded %d CSF subcategories", len(csfDocs))

	// FedRAMP 20x KSI indicators, process requirements, and glossary terms collection
	fedCol, err := db.GetOrCreateCollection("fedramp_20x", nil, embFn)
	if err != nil {
		log.Fatalf("create fedramp_20x collection: %v", err)
	}
	var fedDocs []chromem.Document
	for _, theme := range frmr.KSIThemes {
		for _, ind := range theme.Indicators {
			if content := vec_store.BuildKSIDocument(ind); content != "" {
				fedDocs = append(fedDocs, chromem.Document{
					ID:       ind.ID,
					Content:  content,
					Metadata: map[string]string{"theme_id": ind.ThemeID, "type": "ksi"},
				})
			}
		}
	}
	for _, rc := range frmr.Requirements {
		for _, req := range rc.Requirements {
			if content := vec_store.BuildRequirementDocument(req); content != "" {
				fedDocs = append(fedDocs, chromem.Document{
					ID:       req.ID,
					Content:  content,
					Metadata: map[string]string{"category": req.Category, "type": "requirement"},
				})
			}
		}
	}
	var termCount int
	for _, t := range frmr.Terms {
		if content := vec_store.BuildTermDocument(t); content != "" {
			fedDocs = append(fedDocs, chromem.Document{
				ID:       t.ID,
				Content:  content,
				Metadata: map[string]string{"type": "term"},
			})
			termCount++
		}
	}
	if err := fedCol.AddDocuments(ctx, fedDocs, runtime.NumCPU()); err != nil {
		log.Fatalf("embed fedramp documents: %v", err)
	}
	log.Printf("embedded %d FedRAMP documents (%d terms)", len(fedDocs), termCount)

	dataDir := dataPath()
	if *outputDir != "" {
		abs, err := filepath.Abs(*outputDir)
		if err != nil {
			log.Fatalf("resolve output dir: %v", err)
		}
		dataDir = abs
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("create output dir %s: %v", dataDir, err)
	}

	dbPath := filepath.Join(dataDir, "chromem.db")
	if err := db.ExportToFile(dbPath, true, ""); err != nil {
		log.Fatalf("export chromem DB to %s: %v", dbPath, err)
	}
	log.Printf("wrote %s (%s)", dbPath, fileSize(dbPath))

	meta := vec_store.VectorMeta{
		Provider:         *provider,
		Model:            effectiveModel,
		BuiltAt:          time.Now().UTC(),
		ControlCount:     len(controlDocs),
		MetricCount:      len(fismaDocs),
		SubcategoryCount: len(csfDocs),
		TermCount:        termCount,
	}
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	metaPath := filepath.Join(dataDir, "chromem-meta.json")
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		log.Fatalf("write meta to %s: %v", metaPath, err)
	}
	log.Printf("wrote %s", metaPath)
	log.Println("done — rebuild the binary with 'go build .' to embed the new index")
}

func makeEmbeddingFunc(provider, model, ollamaURL string) (chromem.EmbeddingFunc, string, error) {
	switch provider {
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return nil, "", fmt.Errorf("OPENAI_API_KEY environment variable is required for provider \"openai\"")
		}
		if model == "" {
			model = string(chromem.EmbeddingModelOpenAI3Small)
		}
		return chromem.NewEmbeddingFuncOpenAI(key, chromem.EmbeddingModelOpenAI(model)), model, nil
	case "ollama":
		if model == "" {
			model = "nomic-embed-text"
		}
		return chromem.NewEmbeddingFuncOllama(model, ollamaAPIBase(ollamaURL)), model, nil
	default:
		return nil, "", fmt.Errorf("unknown provider %q (use \"openai\" or \"ollama\")", provider)
	}
}

// dataPath returns the absolute path to internal/nist/data relative to this
// source file so the tool works regardless of where it is invoked from.
func dataPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("cannot determine source file path")
	}
	// file = .../tools/gen-embeddings/main.go
	// data = .../internal/nist/data
	root := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(root, "internal", "vec_store", "data")
}

// ollamaAPIBase ensures the URL has the /api suffix that chromem-go expects.
// Users naturally write http://host:11434; chromem appends /embeddings to whatever it gets.
func ollamaAPIBase(u string) string {
	u = strings.TrimRight(u, "/")
	if !strings.HasSuffix(u, "/api") {
		u += "/api"
	}
	return u
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fileSize(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "?"
	}
	b := info.Size()
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
