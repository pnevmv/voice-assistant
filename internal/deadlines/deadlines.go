// Package deadlines defines the structured deadline record and the LLM-backed
// extractor contract. Wrong dates are the highest-cost error in this system, so
// extraction output is never trusted blindly: see Validate in M2.
package deadlines

import (
	"context"
	"time"
)

// Deadline mirrors the deadlines table.
//
// Date semantics:
//   - single-day event:    StartsOn == EndsOn
//   - "entro il ..." cut-off: StartsOn nil, EndsOn set
//   - period:              StartsOn < EndsOn
//
// DateText is the date exactly as written in the source ("entro il 30 aprile 2027, ore 12:00").
type Deadline struct {
	ID           int64
	VersionID    int64
	ChunkID      *int64
	Event        string // human-readable: "Scadenza domanda borsa di studio"
	EventKey     string // stable slug from a controlled vocabulary
	AppliesTo    string // "all", "matricole", "anni successivi", "studenti internazionali"...
	AcademicYear string
	StartsOn     *time.Time
	EndsOn       *time.Time
	TimeOfDay    string // "12:00" in Europe/Rome, or ""
	DateText     string
	SourceURL    string
	SourceQuote  string // verbatim from the source; must be a substring of the chunk
	QuoteChecked bool
	ReviewStatus string // unreviewed | confirmed | rejected
}

// Candidate is what the LLM returns before validation and persistence.
type Candidate struct {
	Event       string `json:"event"`
	EventKey    string `json:"event_key"`
	AppliesTo   string `json:"applies_to"`
	StartsOn    string `json:"starts_on"` // ISO 8601 or ""
	EndsOn      string `json:"ends_on"`
	TimeOfDay   string `json:"time_of_day"`
	DateText    string `json:"date_text"`
	SourceQuote string `json:"source_quote"`
}

// Extractor runs the extraction prompt over the chunks of one document version.
// It calls the LLM through the gateway with a JSON schema.
type Extractor interface {
	Extract(ctx context.Context, versionID int64) ([]Candidate, error)
}
