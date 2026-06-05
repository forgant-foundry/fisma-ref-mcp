.PHONY: all build-all build build-nomic build-qwen3 build-openai-small build-slim \
        embed-all embed-nomic embed-qwen3 embed-openai-small help

BINARY    := fisma-ref-mcp
GOFLAGS   :=
OLLAMA_URL ?= http://localhost:11434

# ── local builds ────────────────────────────────────────────────────────────

## build: compile using whichever vector index is in data/ (default, untagged)
build:
	go build $(GOFLAGS) -o $(BINARY) .

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
	go build -tags no_embeddings $(GOFLAGS) -o $(BINARY) .

## build-all: compile all four variants to named binaries (fisma-ref-mcp-nomic, etc.)
build-all:
	go build -tags embed_nomic      $(GOFLAGS) -o $(BINARY)-nomic .
	go build -tags embed_qwen3      $(GOFLAGS) -o $(BINARY)-qwen3 .
	go build -tags embed_openai_small $(GOFLAGS) -o $(BINARY)-openai-small .
	go build -tags no_embeddings    $(GOFLAGS) -o $(BINARY)-slim .

# ── index generation (writes to per-model subdirectory, commit the results) ──

## embed-nomic: regenerate data/nomic/ vector index using nomic-embed-text:v1.5 via Ollama
## Requires: Ollama running at OLLAMA_URL with nomic-embed-text:v1.5 pulled
embed-nomic:
	OLLAMA_URL=$(OLLAMA_URL) go run ./tools/gen-embeddings/main.go \
		--provider ollama \
		--model nomic-embed-text:v1.5 \
		--output-dir internal/nist/data/nomic

## embed-qwen3: regenerate data/qwen3/ vector index using qwen3-embedding:4b via Ollama
## Requires: Ollama running at OLLAMA_URL with qwen3-embedding:4b pulled
embed-qwen3:
	OLLAMA_URL=$(OLLAMA_URL) go run ./tools/gen-embeddings/main.go \
		--provider ollama \
		--model qwen3-embedding:4b \
		--output-dir internal/nist/data/qwen3

## embed-openai-small: regenerate data/openai-small/ vector index using text-embedding-3-small
## Requires: OPENAI_API_KEY
embed-openai-small:
	go run ./tools/gen-embeddings/main.go \
		--provider openai \
		--model text-embedding-3-small \
		--output-dir internal/nist/data/openai-small

## embed-all: regenerate all three vector indexes (requires Ollama + OPENAI_API_KEY)
embed-all: embed-nomic embed-qwen3 embed-openai-small

## all: regenerate all indexes then build all variants (requires Ollama + OPENAI_API_KEY)
all: embed-all build-all

help:
	@grep -E '^## ' Makefile | sed 's/^## //'
