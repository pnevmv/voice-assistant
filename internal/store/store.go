// Package store defines retrieval-side storage interfaces. The pgx/pgvector
// implementation lives next to them; sqlc-generated code is in internal/store/db.
package store

import (
	"context"
	"time"
)

// Filter restricts retrieval. The zero value means "current versions only,
// any year" — callers in the RAG service always set AcademicYears explicitly
// (see ADR-0003 and docs/ARCHITECTURE.md §Retrieval).
type Filter struct {
	AcademicYears  []string // e.g. {"2026/27"}; documents with NULL year are matched only if IncludeUndated
	IncludeUndated bool
	DocTypes       []string // bando, avviso, faq, page
	Languages      []string
}

type Hit struct {
	ChunkID int64
	Score   float64 // meaning depends on the searcher; fuse by rank, not by score
}

type ChunkEmbedding struct {
	ChunkID   int64
	Model     string
	Embedding []float32
}

// VectorStore covers the dense side of hybrid retrieval and index writes.
type VectorStore interface {
	UpsertEmbeddings(ctx context.Context, items []ChunkEmbedding) error
	SearchVector(ctx context.Context, query []float32, model string, k int, f Filter) ([]Hit, error)
}

// TextSearcher covers the lexical side (Postgres full-text, Italian config).
type TextSearcher interface {
	SearchText(ctx context.Context, query string, k int, f Filter) ([]Hit, error)
}

// Chunk is a retrieved chunk with the provenance needed for citations.
type Chunk struct {
	ID           int64
	Content      string
	SectionPath  string
	SourceURL    string
	DocType      string
	Language     string
	AcademicYear string
	PublishedAt  *time.Time
	FetchedAt    time.Time // crawl timestamp of the version
}

type ChunkReader interface {
	GetChunks(ctx context.Context, ids []int64) ([]Chunk, error)
}
