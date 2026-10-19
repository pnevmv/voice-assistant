package deadlines

import (
	"fmt"
	"time"

	"unipd-voice-assistant/internal/datex"
	"unipd-voice-assistant/internal/textnorm"
)

// Validation is the outcome of checking one LLM candidate against the text of
// the version it came from. It maps onto deadlines.quote_verified and
// deadlines.date_in_quote; a row is served only when both are true.
type Validation struct {
	QuoteVerified bool
	DateInQuote   bool
	StartsOn      *time.Time
	EndsOn        *time.Time
	Problems      []string
}

// OK reports whether the candidate may be stored as servable.
func (v Validation) OK() bool { return v.QuoteVerified && v.DateInQuote && len(v.Problems) == 0 }

// Validate checks a candidate against versionText (the version's extracted text).
//
//   - The quote must occur verbatim after textnorm.Normalize on both sides — the
//     same normalisation that produces content_hash.
//   - Every date the candidate claims (starts_on / ends_on) must appear as a
//     complete date inside the quote, in any Italian/English/numeric format.
func Validate(c Candidate, versionText string) Validation {
	var v Validation
	problem := func(format string, args ...any) { v.Problems = append(v.Problems, fmt.Sprintf(format, args...)) }

	v.QuoteVerified = textnorm.QuoteInText(versionText, c.SourceQuote)
	if !v.QuoteVerified {
		problem("source_quote not found verbatim in the document text")
	}

	var err error
	if v.StartsOn, err = parseISO(c.StartsOn); err != nil {
		problem("starts_on: %v", err)
	}
	if v.EndsOn, err = parseISO(c.EndsOn); err != nil {
		problem("ends_on: %v", err)
	}
	if v.StartsOn == nil && v.EndsOn == nil {
		problem("no date given")
	}
	if v.StartsOn != nil && v.EndsOn != nil && v.EndsOn.Before(*v.StartsOn) {
		problem("ends_on is before starts_on")
	}

	v.DateInQuote = (v.StartsOn != nil || v.EndsOn != nil)
	for _, d := range []*time.Time{v.StartsOn, v.EndsOn} {
		if d != nil && !datex.Contains(c.SourceQuote, *d) {
			v.DateInQuote = false
			problem("date %s not found in the quote", d.Format(time.DateOnly))
		}
	}
	return v
}

func parseISO(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return nil, fmt.Errorf("%q is not an ISO date", s)
	}
	return &t, nil
}
