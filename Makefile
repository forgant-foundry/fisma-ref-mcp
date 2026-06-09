.PHONY: build-all build-nomic build-qwen3 build-openai-small build-slim build-onnx \
        embed-all embed-nomic embed-qwen3 embed-openai-small embed-onnx \
        fetch-onnx-artifacts clean help

BINARY             := fisma-ref-mcp
GOFLAGS            :=
OLLAMA_URL         ?= http://localhost:11434
TOKENIZERS_VERSION := 1.27.0
ONNX_DATA_DIR      := internal/vec_store/data/onnx
LIBTOKENIZERS_DIR  := $(ONNX_DATA_DIR)/lib

# ── local builds ────────────────────────────────────────────────────────────

## build-nomic: compile with the nomic-embed-text:v1.5 vector index
build-nomic:
	go build -tags embed_nomic $(GOFLAGS) -o $(BINARY) .

## build-qwen3: compile with the qwen3-embedding:4b vector index
build-qwen3:
	go build -tags embed_qwen3 $(GOFLAGS) -o $(BINARY) .

## build-openai-small: compile with the text-embedding-3-small vector index
build-openai-small:
	go build -tags embed_openai_small $(GOFLAGS) -o $(BINARY) .

## build-slim: compile without any vector index; FTS5 search only
build-slim:
	go build $(GOFLAGS) -o $(BINARY) .

## build-onnx: compile with the ONNX all-MiniLM-L6-v2 INT8 vector index (CGo required)
## Requires: ONNX Runtime installed (brew install onnxruntime on macOS)
## Requires: libtokenizers.a present (run make fetch-onnx-artifacts first)
## Requires: data/onnx/chromem.db present (run make embed-onnx first)
build-onnx:
	CGO_ENABLED=1 CGO_LDFLAGS="-L$(CURDIR)/$(LIBTOKENIZERS_DIR)" \
		go build -tags embed_onnx $(GOFLAGS) -o $(BINARY)-onnx .

## build-all: compile all four pure-Go variants to named binaries (fisma-ref-mcp-nomic, etc.)
## Note: build-onnx (CGo) is intentionally excluded; run it separately.
build-all:
	go build -tags embed_nomic        $(GOFLAGS) -o $(BINARY)-nomic .
	go build -tags embed_qwen3        $(GOFLAGS) -o $(BINARY)-qwen3 .
	go build -tags embed_openai_small $(GOFLAGS) -o $(BINARY)-openai-small .
	go build                          $(GOFLAGS) -o $(BINARY)-slim .

# ── index generation (writes to per-model subdirectory, commit the results) ──

## embed-nomic: regenerate data/nomic/ vector index using nomic-embed-text:v1.5 via Ollama
## Requires: Ollama running at OLLAMA_URL with nomic-embed-text:v1.5 pulled
embed-nomic:
	OLLAMA_URL=$(OLLAMA_URL) go run ./tools/gen-embeddings/main.go \
		--provider ollama \
		--model nomic-embed-text:v1.5 \
		--output-dir internal/vec_store/data/nomic

## embed-qwen3: regenerate data/qwen3/ vector index using qwen3-embedding:4b via Ollama
## Requires: Ollama running at OLLAMA_URL with qwen3-embedding:4b pulled
embed-qwen3:
	OLLAMA_URL=$(OLLAMA_URL) go run ./tools/gen-embeddings/main.go \
		--provider ollama \
		--model qwen3-embedding:4b \
		--output-dir internal/vec_store/data/qwen3

## embed-openai-small: regenerate data/openai-small/ vector index using text-embedding-3-small
## Requires: OPENAI_API_KEY
embed-openai-small:
	go run ./tools/gen-embeddings/main.go \
		--provider openai \
		--model text-embedding-3-small \
		--output-dir internal/vec_store/data/openai-small

## embed-all: regenerate all three pure-Go vector indexes (requires Ollama + OPENAI_API_KEY)
embed-all: embed-nomic embed-qwen3 embed-openai-small

## fetch-onnx-artifacts: download ONNX model, tokenizer, and libtokenizers.a for the current platform
fetch-onnx-artifacts:
	mkdir -p $(LIBTOKENIZERS_DIR)
	@if [ ! -s $(ONNX_DATA_DIR)/tokenizer.json ]; then \
		echo "downloading tokenizer.json..."; \
		curl -fSL -o $(ONNX_DATA_DIR)/tokenizer.json \
			"https://huggingface.co/Xenova/all-MiniLM-L6-v2/resolve/main/tokenizer.json"; \
	fi
	@if [ ! -s $(ONNX_DATA_DIR)/model_int8.onnx ]; then \
		echo "downloading model_int8.onnx (~23 MB)..."; \
		curl -fSL -o $(ONNX_DATA_DIR)/model_int8.onnx \
			"https://huggingface.co/Xenova/all-MiniLM-L6-v2/resolve/main/onnx/model_int8.onnx"; \
	fi
	@if [ ! -f $(LIBTOKENIZERS_DIR)/libtokenizers.a ]; then \
		echo "downloading libtokenizers.a..."; \
		OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		ARCH=$$(uname -m | sed 's/arm64/aarch64/'); \
		curl -fSL -o /tmp/libtokenizers.tar.gz \
			"https://github.com/daulet/tokenizers/releases/download/v$(TOKENIZERS_VERSION)/libtokenizers.$${OS}-$${ARCH}.tar.gz" && \
		tar -xzf /tmp/libtokenizers.tar.gz -C $(LIBTOKENIZERS_DIR); \
	fi

## embed-onnx: regenerate data/onnx/ vector index using ONNX all-MiniLM-L6-v2 (CGo required)
## Requires: ONNX Runtime installed, libtokenizers.a present (run fetch-onnx-artifacts first)
## After running, commit the updated internal/vec_store/data/onnx/ directory.
embed-onnx: fetch-onnx-artifacts
	CGO_ENABLED=1 CGO_LDFLAGS="-L$(CURDIR)/$(LIBTOKENIZERS_DIR)" \
		go run ./tools/gen-embeddings-onnx/main.go \
			--model-path $(ONNX_DATA_DIR)/model_int8.onnx \
			--tokenizer  $(ONNX_DATA_DIR)/tokenizer.json \
			--output-dir $(ONNX_DATA_DIR)

## parse-fisma: (re)parse the FY 2025 IG FISMA Metrics PDF → internal/fisma/data/fy2025-ig-fisma-metrics.json
parse-fisma:
	python3 tools/parse-fisma-metrics/parse.py

## clean: remove compiled binaries
clean:
	rm -f $(BINARY) $(BINARY)-nomic $(BINARY)-qwen3 $(BINARY)-openai-small $(BINARY)-slim $(BINARY)-onnx

help:
	@grep -E '^## ' Makefile | sed 's/^## //'
