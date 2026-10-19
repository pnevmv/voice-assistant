// Package embed defines the embedding abstraction. The production
// implementation calls the LLM gateway's /v1/embeddings (bge-m3 via Ollama by
// default), so embeddings obey the "LLMs only through the gateway" rule.
package embed

import "context"

// Kind distinguishes queries from documents: e5-style models need different
// prefixes, and some providers expose separate modes.
type Kind int

const (
	KindDocument Kind = iota
	KindQuery
)

type Embedder interface {
	// Model is the identifier stored in chunks.embedding_model. Vectors from
	// different models must never be compared.
	Model() string
	// Dimensions must match the chunks.embedding column (1024 initially).
	Dimensions() int
	// Embed returns one vector per input, in order. Implementations batch
	// internally and are expected to return L2-normalised vectors.
	Embed(ctx context.Context, texts []string, kind Kind) ([][]float32, error)
}
