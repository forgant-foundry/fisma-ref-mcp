//go:build !embed_onnx

package vec_store

import (
	"fmt"

	chromem "github.com/philippgille/chromem-go"
)

func newONNXEmbeddingFunc() (chromem.EmbeddingFunc, error) {
	return nil, fmt.Errorf("ONNX embedding not compiled in this build; rebuild with -tags embed_onnx")
}
