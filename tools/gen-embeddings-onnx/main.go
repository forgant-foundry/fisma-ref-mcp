//go:build ignore

// gen-embeddings-onnx builds the chromem-go vector index using the ONNX
// all-MiniLM-L6-v2 model running in-process via onnxruntime_go.  It writes
// chromem.db and chromem-meta.json into the output directory so they can be
// compiled into the binary with //go:embed.
//
// Prerequisites:
//
//	brew install onnxruntime                     (macOS)
//	sudo ldconfig /usr/local/lib                 (Linux, after installing ONNX Runtime)
//
//	# Download libtokenizers.a and set CGO_LDFLAGS before running:
//	make fetch-onnx-artifacts
//
// Usage (via Makefile):
//
//	make embed-onnx
//
// Or directly:
//
//	CGO_ENABLED=1 CGO_LDFLAGS="-L./internal/vec_store/data/onnx/lib" \
//	  go run ./tools/gen-embeddings-onnx/main.go \
//	    --model-path  internal/vec_store/data/onnx/model_int8.onnx \
//	    --tokenizer   internal/vec_store/data/onnx/tokenizer.json \
//	    --output-dir  internal/vec_store/data/onnx
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tokenizers "github.com/daulet/tokenizers"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/fedramp"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/fisma"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_800_53"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/nist_csf"
	"github.com/forgant-foundry/fisma-ref-mcp/internal/vec_store"
	chromem "github.com/philippgille/chromem-go"
	ort "github.com/yalue/onnxruntime_go"
)

const (
	maxSeqLen = 512
	embedDim  = 384
	modelName = "all-MiniLM-L6-v2-int8"
)

func main() {
	modelPath := flag.String("model-path", "", "Path to model_int8.onnx (required)")
	tokenizerPath := flag.String("tokenizer", "", "Path to tokenizer.json (required)")
	outputDir := flag.String("output-dir", "", "Output directory for chromem.db and chromem-meta.json (required)")
	flag.Parse()

	if *modelPath == "" || *tokenizerPath == "" || *outputDir == "" {
		flag.Usage()
		log.Fatal("--model-path, --tokenizer, and --output-dir are all required")
	}

	embFn, err := newONNXEmbeddingFunc(*modelPath, *tokenizerPath)
	if err != nil {
		log.Fatalf("init ONNX embedder: %v", err)
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

	log.Printf("indexing %d controls + %d FISMA metrics + %d CSF subcategories + FedRAMP with onnx/%s ...",
		len(controls), len(metrics), len(subcategories), modelName)

	db := chromem.NewDB()

	controlCol, err := db.GetOrCreateCollection("controls", nil, embFn)
	if err != nil {
		log.Fatalf("create controls collection: %v", err)
	}
	var controlDocs []chromem.Document
	for _, c := range controls {
		if content := vec_store.BuildControlDocument(c); content != "" {
			controlDocs = append(controlDocs, chromem.Document{
				ID:       strings.ToUpper(c.ID),
				Content:  content,
				Metadata: map[string]string{"family": c.FamilyID, "is_enhancement": fmt.Sprintf("%v", c.IsEnhancement)},
			})
		}
	}
	if err := controlCol.AddDocuments(ctx, controlDocs, runtime.NumCPU()); err != nil {
		log.Fatalf("embed controls: %v", err)
	}
	log.Printf("embedded %d controls", len(controlDocs))

	fismaCol, err := db.GetOrCreateCollection("fisma_metrics", nil, embFn)
	if err != nil {
		log.Fatalf("create fisma_metrics collection: %v", err)
	}
	var fismaDocs []chromem.Document
	for _, m := range metrics {
		if content := vec_store.BuildMetricDocument(m); content != "" {
			fismaDocs = append(fismaDocs, chromem.Document{
				ID:       fmt.Sprintf("%d", m.ID),
				Content:  content,
				Metadata: map[string]string{"domain": m.Domain, "review_cycle": m.ReviewCycle},
			})
		}
	}
	if err := fismaCol.AddDocuments(ctx, fismaDocs, runtime.NumCPU()); err != nil {
		log.Fatalf("embed fisma metrics: %v", err)
	}
	log.Printf("embedded %d FISMA metrics", len(fismaDocs))

	csfCol, err := db.GetOrCreateCollection("csf_v2", nil, embFn)
	if err != nil {
		log.Fatalf("create csf_v2 collection: %v", err)
	}
	var csfDocs []chromem.Document
	for _, s := range subcategories {
		if content := vec_store.BuildSubcategoryDocument(s); content != "" {
			csfDocs = append(csfDocs, chromem.Document{
				ID:       s.ID,
				Content:  content,
				Metadata: map[string]string{"category_id": s.CategoryID, "function_id": s.FunctionID},
			})
		}
	}
	if err := csfCol.AddDocuments(ctx, csfDocs, runtime.NumCPU()); err != nil {
		log.Fatalf("embed CSF subcategories: %v", err)
	}
	log.Printf("embedded %d CSF subcategories", len(csfDocs))

	fedCol, err := db.GetOrCreateCollection("fedramp_20x", nil, embFn)
	if err != nil {
		log.Fatalf("create fedramp_20x collection: %v", err)
	}
	var fedDocs []chromem.Document
	var termCount int
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
		log.Fatalf("embed FedRAMP documents: %v", err)
	}
	log.Printf("embedded %d FedRAMP documents (%d terms)", len(fedDocs), termCount)

	abs, err := filepath.Abs(*outputDir)
	if err != nil {
		log.Fatalf("resolve output dir: %v", err)
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	dbPath := filepath.Join(abs, "chromem.db")
	if err := db.ExportToFile(dbPath, true, ""); err != nil {
		log.Fatalf("export chromem DB: %v", err)
	}
	log.Printf("wrote %s (%s)", dbPath, fileSize(dbPath))

	meta := vec_store.VectorMeta{
		Provider:         "onnx",
		Model:            modelName,
		BuiltAt:          time.Now().UTC(),
		ControlCount:     len(controlDocs),
		MetricCount:      len(fismaDocs),
		SubcategoryCount: len(csfDocs),
		TermCount:        termCount,
	}
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	metaPath := filepath.Join(abs, "chromem-meta.json")
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		log.Fatalf("write meta: %v", err)
	}
	log.Printf("wrote %s", metaPath)
	log.Println("done — rebuild with: go build -tags embed_onnx .")
}

// newONNXEmbeddingFunc creates a chromem.EmbeddingFunc backed by the
// in-process ONNX Runtime using the given model and tokenizer files.
func newONNXEmbeddingFunc(modelPath, tokenizerPath string) (chromem.EmbeddingFunc, error) {
	modelBytes, err := os.ReadFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("read model: %w", err)
	}
	tokenizerBytes, err := os.ReadFile(tokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("read tokenizer: %w", err)
	}

	if lib := onnxLibPath(); lib != "" {
		ort.SetSharedLibraryPath(lib)
	}
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("init ONNX Runtime: %w", err)
	}

	tk, err := tokenizers.FromBytesWithTruncation(tokenizerBytes, maxSeqLen, tokenizers.TruncationDirectionRight)
	if err != nil {
		return nil, fmt.Errorf("load tokenizer: %w", err)
	}

	session, err := ort.NewDynamicAdvancedSessionWithONNXData(
		modelBytes,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"last_hidden_state"},
		nil,
	)
	if err != nil {
		tk.Close()
		return nil, fmt.Errorf("create ONNX session: %w", err)
	}

	return func(_ context.Context, text string) ([]float32, error) {
		enc := tk.EncodeWithOptions(text, true,
			tokenizers.WithReturnTypeIDs(),
			tokenizers.WithReturnAttentionMask(),
		)
		seqLen := len(enc.IDs)
		if seqLen == 0 {
			return nil, fmt.Errorf("empty encoding")
		}

		ids := make([]int64, seqLen)
		mask := make([]int64, seqLen)
		typeIDs := make([]int64, seqLen)
		for i, v := range enc.IDs {
			ids[i] = int64(v)
		}
		for i, v := range enc.AttentionMask {
			mask[i] = int64(v)
		}
		for i, v := range enc.TypeIDs {
			typeIDs[i] = int64(v)
		}

		shape := ort.NewShape(1, int64(seqLen))
		idsTensor, err := ort.NewTensor(shape, ids)
		if err != nil {
			return nil, err
		}
		defer idsTensor.Destroy()
		maskTensor, err := ort.NewTensor(shape, mask)
		if err != nil {
			return nil, err
		}
		defer maskTensor.Destroy()
		typeIDsTensor, err := ort.NewTensor(shape, typeIDs)
		if err != nil {
			return nil, err
		}
		defer typeIDsTensor.Destroy()

		outputs := []ort.Value{nil}
		if err := session.Run([]ort.Value{idsTensor, maskTensor, typeIDsTensor}, outputs); err != nil {
			return nil, fmt.Errorf("ONNX inference: %w", err)
		}
		if outputs[0] == nil {
			return nil, fmt.Errorf("nil output from ONNX inference")
		}
		defer outputs[0].Destroy()

		floatTensor, ok := outputs[0].(*ort.Tensor[float32])
		if !ok {
			return nil, fmt.Errorf("unexpected output type")
		}
		hidden := floatTensor.GetData()
		return l2Normalize(meanPooling(hidden, mask, seqLen)), nil
	}, nil
}

func meanPooling(hidden []float32, mask []int64, seqLen int) []float32 {
	out := make([]float32, embedDim)
	var maskSum int64
	for i := 0; i < seqLen; i++ {
		if mask[i] == 0 {
			continue
		}
		maskSum += mask[i]
		for j := 0; j < embedDim; j++ {
			out[j] += hidden[i*embedDim+j] * float32(mask[i])
		}
	}
	if maskSum > 0 {
		s := float32(maskSum)
		for j := range out {
			out[j] /= s
		}
	}
	return out
}

func l2Normalize(v []float32) []float32 {
	var norm float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	if n := math.Sqrt(norm); n > 0 {
		for i, x := range v {
			v[i] = float32(float64(x) / n)
		}
	}
	return v
}

func onnxLibPath() string {
	if v := os.Getenv("ONNX_RUNTIME_LIB"); v != "" {
		return v
	}
	switch runtime.GOOS {
	case "darwin":
		if _, err := os.Stat("/opt/homebrew/lib/libonnxruntime.dylib"); err == nil {
			return "/opt/homebrew/lib/libonnxruntime.dylib"
		}
		if _, err := os.Stat("/usr/local/lib/libonnxruntime.dylib"); err == nil {
			return "/usr/local/lib/libonnxruntime.dylib"
		}
	case "linux":
		for _, p := range []string{
			"/usr/local/lib/libonnxruntime.so",
			"/usr/lib/libonnxruntime.so",
			"/usr/lib/x86_64-linux-gnu/libonnxruntime.so",
			"/usr/lib/aarch64-linux-gnu/libonnxruntime.so",
		} {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
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
