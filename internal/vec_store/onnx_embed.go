//go:build embed_onnx

package vec_store

import (
	"context"
	"fmt"
	"math"
	"os"
	"runtime"
	"sync"

	tokenizers "github.com/daulet/tokenizers"
	chromem "github.com/philippgille/chromem-go"
	ort "github.com/yalue/onnxruntime_go"
)

const (
	onnxMaxSeqLen = 512
	onnxEmbedDim  = 384
)

var (
	globalEmbedder     *onnxEmbedder
	globalEmbedderErr  error
	globalEmbedderOnce sync.Once
)

type onnxEmbedder struct {
	tk      *tokenizers.Tokenizer
	session *ort.DynamicAdvancedSession
}

func getGlobalEmbedder() (*onnxEmbedder, error) {
	globalEmbedderOnce.Do(func() {
		globalEmbedder, globalEmbedderErr = initONNXEmbedder()
	})
	return globalEmbedder, globalEmbedderErr
}

func initONNXEmbedder() (*onnxEmbedder, error) {
	if len(onnxModelBytes) == 0 {
		return nil, fmt.Errorf("ONNX model not embedded: run 'make embed-onnx' then rebuild with -tags embed_onnx")
	}
	if len(onnxTokenizerJSON) == 0 {
		return nil, fmt.Errorf("tokenizer not embedded: run 'make embed-onnx' then rebuild with -tags embed_onnx")
	}

	if lib := onnxLibraryPath(); lib != "" {
		ort.SetSharedLibraryPath(lib)
	}
	if err := ort.InitializeEnvironment(); err != nil {
		return nil, fmt.Errorf("init ONNX Runtime: %w", err)
	}

	tk, err := tokenizers.FromBytesWithTruncation(onnxTokenizerJSON, onnxMaxSeqLen, tokenizers.TruncationDirectionRight)
	if err != nil {
		return nil, fmt.Errorf("load tokenizer: %w", err)
	}

	session, err := ort.NewDynamicAdvancedSessionWithONNXData(
		onnxModelBytes,
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"last_hidden_state"},
		nil,
	)
	if err != nil {
		tk.Close()
		return nil, fmt.Errorf("create ONNX session: %w", err)
	}

	return &onnxEmbedder{tk: tk, session: session}, nil
}

func newONNXEmbeddingFunc() (chromem.EmbeddingFunc, error) {
	emb, err := getGlobalEmbedder()
	if err != nil {
		return nil, err
	}
	return func(_ context.Context, text string) ([]float32, error) {
		return emb.embed(text)
	}, nil
}

func (e *onnxEmbedder) embed(text string) ([]float32, error) {
	enc := e.tk.EncodeWithOptions(text, true,
		tokenizers.WithReturnTypeIDs(),
		tokenizers.WithReturnAttentionMask(),
	)

	seqLen := len(enc.IDs)
	if seqLen == 0 {
		return nil, fmt.Errorf("tokenizer produced empty encoding")
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
		return nil, fmt.Errorf("create input_ids tensor: %w", err)
	}
	defer idsTensor.Destroy()

	maskTensor, err := ort.NewTensor(shape, mask)
	if err != nil {
		return nil, fmt.Errorf("create attention_mask tensor: %w", err)
	}
	defer maskTensor.Destroy()

	typeIDsTensor, err := ort.NewTensor(shape, typeIDs)
	if err != nil {
		return nil, fmt.Errorf("create token_type_ids tensor: %w", err)
	}
	defer typeIDsTensor.Destroy()

	// Pass nil output; the runtime allocates the last_hidden_state tensor.
	outputs := []ort.Value{nil}
	if err := e.session.Run(
		[]ort.Value{idsTensor, maskTensor, typeIDsTensor},
		outputs,
	); err != nil {
		return nil, fmt.Errorf("ONNX inference: %w", err)
	}
	if outputs[0] == nil {
		return nil, fmt.Errorf("ONNX inference returned nil output")
	}
	defer outputs[0].Destroy()

	floatTensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected ONNX output type (expected *Tensor[float32])")
	}
	// hidden: flat [1, seqLen, onnxEmbedDim]
	hidden := floatTensor.GetData()

	return normalizeEmbedding(meanPool(hidden, mask, seqLen)), nil
}

// meanPool computes attention-mask-weighted mean pooling over the sequence dimension.
func meanPool(hidden []float32, mask []int64, seqLen int) []float32 {
	out := make([]float32, onnxEmbedDim)
	var maskSum int64
	for i := 0; i < seqLen; i++ {
		if mask[i] == 0 {
			continue
		}
		maskSum += mask[i]
		for j := 0; j < onnxEmbedDim; j++ {
			out[j] += hidden[i*onnxEmbedDim+j] * float32(mask[i])
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

// normalizeEmbedding applies L2 normalization in-place and returns v.
func normalizeEmbedding(v []float32) []float32 {
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

// onnxLibraryPath resolves the ONNX Runtime shared library path.
// Checks ONNX_RUNTIME_LIB first, then platform-specific defaults.
// Returns "" to let onnxruntime_go use its built-in default ("onnxruntime.so"),
// which works on Linux when ldconfig has been run after installation.
func onnxLibraryPath() string {
	if v := os.Getenv("ONNX_RUNTIME_LIB"); v != "" {
		return v
	}
	switch runtime.GOOS {
	case "darwin":
		// Homebrew ARM64 (Apple Silicon)
		if _, err := os.Stat("/opt/homebrew/lib/libonnxruntime.dylib"); err == nil {
			return "/opt/homebrew/lib/libonnxruntime.dylib"
		}
		// Homebrew x86_64 (Intel)
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
