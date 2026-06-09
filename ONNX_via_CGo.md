# ONNX Embedding via CGo — Fifth Build Target

Design notes for adding a fully local, network-free embedding variant using ONNX Runtime
in-process. Intended for environments where Ollama and OpenAI are not permitted by governance.

## Motivation

The four existing build targets all require an external embedding service at query time:

| Target | Runtime embedding dependency |
|---|---|
| `embed_nomic` | Ollama (local server process) |
| `embed_qwen3` | Ollama (local server process) |
| `embed_openai_small` | OpenAI API (network) |
| slim (no tag) | None — FTS5 keyword search only |

The ONNX variant (`embed_onnx`) eliminates all external dependencies at query time by running
the embedding model in-process via the ONNX Runtime C library. No network calls, no separate
process to install or manage.

## How it fits the existing architecture

The pattern is identical to the other `embed_*` variants:

- A new `internal/vec_store/embed_onnx.go` file embeds the pre-built `chromem.db` index, the
  `chromem-meta.json`, the `.onnx` model file, and the tokenizer vocab into the binary using
  `//go:embed`.
- `internal/vec_store/embed_stub.go` gains `!embed_onnx` in its build constraint.
- `internal/vec_store/vector.go` gains an `"onnx"` case in `embeddingFunc()` using
  `onnxruntime_go` + tokenizer bindings.
- `tools/gen-embeddings/main.go` gains a `--provider onnx` flag so the index is built with
  the same model used at query time.
- `Makefile` gains `build-onnx` and `embed-onnx` targets.

One architectural difference from the other variants: the ONNX model must be available at
**both** dev index-build time (`embed-onnx`) and at runtime (embedded in the binary). The
other variants call an external API at both times; this variant eliminates the runtime call by
shipping the model in the binary.

## Model options

The recommended base model is `all-MiniLM-L6-v2`, a sentence transformer that produces
384-dimensional embeddings. Pre-exported ONNX variants are available from HuggingFace
([Xenova/all-MiniLM-L6-v2](https://huggingface.co/Xenova/all-MiniLM-L6-v2/tree/main/onnx))
with no conversion step required.

| Variant | File | Binary size contribution | Notes |
|---|---|---|---|
| FP32 | `model.onnx` | ~90 MB | Full precision |
| FP16 | `model_fp16.onnx` | ~45 MB | Half precision |
| INT8 | `model_int8.onnx` | ~23 MB | Recommended |

**INT8 recommendation:** `all-MiniLM-L6-v2` is unusually resilient to INT8 quantization.
Quality degradation for semantic similarity tasks is less than 1–2%, well below the 4%+ loss
typical of other models. Vector space integrity is preserved well enough for retrieval and
ranking. The 4x size reduction and 2–4x inference speedup make INT8 the practical default.

The final binary size using `//go:embed` is approximately:

- FP32: binary + ~93 MB (model + chromem.db index)
- INT8: binary + ~26 MB (model + chromem.db index)

## Go library dependencies

Two CGo libraries are required:

### `github.com/yalue/onnxruntime_go`

Wraps the official ONNX Runtime C++ library. The Go library links dynamically — the ONNX
Runtime shared library must be present on the machine at runtime (see below).

### `github.com/daulet/tokenizers`

Wraps the Rust-based HuggingFace tokenizers library. Pre-built static archives (`libtokenizers.a`)
are available from the [project releases](https://github.com/daulet/tokenizers/releases) for:
`darwin-arm64`, `darwin-x86_64`, `linux-amd64`, `linux-arm64`, `linux-s390x`, `linux-ppc64le`.
This is a **compile-time** static link — no side file required at runtime.

### Pure-Go tokenization alternative

If the `daulet/tokenizers` CGo dependency is undesirable, `github.com/sugarme/tokenizer`
(pure Go, WordPiece support) is a candidate, but compatibility with all-MiniLM-L6-v2's
`tokenizer.json` format requires verification before committing to it.

## What users need in addition to the binary

Unlike all other build targets, the `embed_onnx` binary has **one runtime side-file
requirement**: the ONNX Runtime shared library.

| Platform | File |
|---|---|
| macOS | `libonnxruntime.dylib` |
| Linux | `libonnxruntime.so.1` |
| Windows | `onnxruntime.dll` |

The version must match what `onnxruntime_go` was compiled against (check the library's
`go.mod` or release notes for the required version).

**macOS**

```sh
brew install onnxruntime
```

**Linux**

Download the pre-built archive from [microsoft/onnxruntime releases](https://github.com/microsoft/onnxruntime/releases),
then install the `.so` to a system library path:

```sh
VERSION=1.20.1   # use the version matching your onnxruntime_go dependency
wget https://github.com/microsoft/onnxruntime/releases/download/v${VERSION}/onnxruntime-linux-x64-${VERSION}.tgz
tar -xzf onnxruntime-linux-x64-${VERSION}.tgz
sudo cp onnxruntime-linux-x64-${VERSION}/lib/libonnxruntime.so.${VERSION} /usr/local/lib/
sudo ldconfig
```

Substitute `linux-aarch64` for `linux-x64` on ARM hosts.

**Windows**

Download the pre-built zip from [microsoft/onnxruntime releases](https://github.com/microsoft/onnxruntime/releases),
extract it, and place `onnxruntime.dll` either alongside the binary or in a directory on `PATH`:

```powershell
# Example using PowerShell (adjust VERSION and arch as needed)
$VERSION = "1.20.1"
Invoke-WebRequest `
  "https://github.com/microsoft/onnxruntime/releases/download/v$VERSION/onnxruntime-win-x64-$VERSION.zip" `
  -OutFile onnxruntime.zip
Expand-Archive onnxruntime.zip -DestinationPath onnxruntime
# Copy onnxruntime.dll next to fisma-ref-mcp.exe, or add the lib\ folder to PATH
```

The ONNX model and tokenizer vocab are embedded in the binary itself — users do not need to
download or manage the model file separately.

## CGo implications

CGo is opt-in per build tag. The `embed_onnx` target requires:

**At build time:**
- A C compiler (gcc or clang)
- `libtokenizers.a` for the target platform (from daulet/tokenizers releases)
- `CGO_LDFLAGS` pointing to the directory containing `libtokenizers.a`

**At runtime:**
- The ONNX Runtime shared library (version must match what `onnxruntime_go` was compiled against)

The slim, nomic, qwen3, and openai-small targets are unaffected and remain pure-Go builds.

## Build target summary (all five)

| Make target | Tag | Embedding at query time | External runtime dep |
|---|---|---|---|
| `build-slim` | none | FTS5 keyword search | None |
| `build-nomic` | `embed_nomic` | Ollama (nomic-embed-text:v1.5) | Ollama process |
| `build-qwen3` | `embed_qwen3` | Ollama (qwen3-embedding:4b) | Ollama process |
| `build-openai-small` | `embed_openai_small` | OpenAI API | Network + API key |
| `build-onnx` | `embed_onnx` | In-process ONNX Runtime | ONNX Runtime shared lib |
