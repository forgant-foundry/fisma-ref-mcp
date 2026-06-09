//go:build embed_onnx

package vec_store

import _ "embed"

//go:embed data/onnx/chromem.db
var prebuiltVectorDB []byte

//go:embed data/onnx/chromem-meta.json
var prebuiltVectorMetaJSON []byte

//go:embed data/onnx/model_int8.onnx
var onnxModelBytes []byte

//go:embed data/onnx/tokenizer.json
var onnxTokenizerJSON []byte

// PrebuiltVector returns the serialised chromem-go DB and its metadata.
func PrebuiltVector() ([]byte, *VectorMeta, bool) {
	return decodePrebuilt(prebuiltVectorDB, prebuiltVectorMetaJSON)
}
