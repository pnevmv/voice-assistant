// Package rag holds the RAG service's public contract. The wire format is
// specified in docs/ARCHITECTURE.md §API contracts; keep both in sync.
package rag

import (
	"context"
	"iter"
	"time"
)

type Status string

const (
	// StatusAnswered: the answer is grounded in at least one cited source.
	StatusAnswered Status = "answered"
	// StatusNotFound: nothing relevant in the official documents. The answer
	// text says so explicitly instead of guessing.
	StatusNotFound Status = "not_found"
	// StatusPartial: some parts of the question could be answered, some not.
	StatusPartial Status = "partial"
)

type AskRequest struct {
	Question     string `json:"question"`
	Language     string `json:"language,omitempty"`      // "" = detect
	AcademicYear string `json:"academic_year,omitempty"` // "" = current year from config
	Stream       bool   `json:"stream,omitempty"`
}

// Citation is what every answer must carry. Dates are what the user needs to
// judge freshness: PublishedAt is the document's own date, CrawledAt is when we
// last saw it.
type Citation struct {
	N       int    `json:"n"` // matches [n] markers in the answer text
	URL     string `json:"url"`
	Title   string `json:"title"`
	DocType string `json:"doc_type"`
	// InformationalTranslation marks an English (or other) translation of an
	// Italian original: it is cited after the original and is not legally valid
	// (ADR-0010). OriginalURL then points at the Italian document.
	InformationalTranslation bool       `json:"informational_translation,omitempty"`
	OriginalURL              string     `json:"original_url,omitempty"`
	AcademicYear             string     `json:"academic_year,omitempty"`
	Section                  string     `json:"section,omitempty"` // "Art. 7"
	PublishedAt              *time.Time `json:"published_at,omitempty"`
	CrawledAt                time.Time  `json:"crawled_at"`
	Quote                    string     `json:"quote,omitempty"`
}

type DeadlineRef struct {
	Event        string `json:"event"`
	AppliesTo    string `json:"applies_to"`
	StartsOn     string `json:"starts_on,omitempty"` // ISO date
	EndsOn       string `json:"ends_on,omitempty"`
	TimeOfDay    string `json:"time_of_day,omitempty"`
	DateText     string `json:"date_text"`
	AcademicYear string `json:"academic_year"`
	SourceURL    string `json:"source_url"`
	SourceQuote  string `json:"source_quote"`
	Reviewed     bool   `json:"reviewed"` // true only when review_status = confirmed
}

type Answer struct {
	Text         string `json:"answer"`
	Language     string `json:"language"`
	Status       Status `json:"status"`
	AcademicYear string `json:"academic_year"`
	AsOf         string `json:"as_of"` // date the answer was produced (ISO)
	// OldestSource is the oldest crawl time (fetched_at) among the cited sources.
	OldestSource *time.Time    `json:"oldest_source_crawled_at,omitempty"`
	Citations    []Citation    `json:"citations"`
	Deadlines    []DeadlineRef `json:"deadlines,omitempty"`
	TraceID      string        `json:"trace_id,omitempty"`
}

// Event is one SSE event of a streaming answer: "meta", "delta", "done", "error".
type Event struct {
	Kind string
	Data any
}

type Service interface {
	Ask(ctx context.Context, req AskRequest) (*Answer, error)
	AskStream(ctx context.Context, req AskRequest) iter.Seq2[Event, error]
}
