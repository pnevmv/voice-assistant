package speech

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// ElevenLabs `keyterms` limits (supported by scribe_v3, scribe_v3_live, scribe_v2),
// as verified in the ElevenLabs docs on 2026-11-24.
const (
	MaxTermChars = 50 // per term, batch and realtime
	MaxTermWords = 5  // per term, batch and realtime

	MaxBatchTerms = 1000 // batch requests

	MaxRealtimeTerms = 50   // realtime: scribe_v3_live, repeated `keyterms` query params
	MaxRealtimeChars = 1000 // realtime: total characters over all terms
)

// LoadKeyterms reads a keyterms file (one term per line, '#' comments, blank
// lines ignored), drops case-insensitive duplicates keeping the first spelling,
// and validates the result against the REALTIME limits. Validating against the
// stricter realtime limits means a switch from scribe_v3 (batch) to
// scribe_v3_live needs no change to the list. Intended to run at startup: an
// invalid list is a configuration error, not something to discover mid-call.
func LoadKeyterms(path string) ([]string, error) {
	f, err := os.Open(path) //nolint:gosec // path comes from operator-controlled configuration
	if err != nil {
		return nil, fmt.Errorf("keyterms: %w", err)
	}
	defer func() { _ = f.Close() }() // read-only file

	var terms []string
	seen := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if key := strings.ToLower(line); !seen[key] {
			seen[key] = true
			terms = append(terms, line)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("keyterms: read %s: %w", path, err)
	}
	if err := ValidateKeyterms(terms, true); err != nil {
		return nil, fmt.Errorf("keyterms %s: %w", path, err)
	}
	return terms, nil
}

// ValidateKeyterms checks terms against the batch or realtime limits and
// reports every violation, not only the first.
func ValidateKeyterms(terms []string, realtime bool) error {
	var errs []error
	maxTerms := MaxBatchTerms
	if realtime {
		maxTerms = MaxRealtimeTerms
	}
	if len(terms) > maxTerms {
		errs = append(errs, fmt.Errorf("%d terms, limit is %d", len(terms), maxTerms))
	}
	total := 0
	for _, t := range terms {
		n := utf8.RuneCountInString(t)
		total += n
		if n > MaxTermChars {
			errs = append(errs, fmt.Errorf("term %q has %d characters, limit is %d", t, n, MaxTermChars))
		}
		if w := len(strings.Fields(t)); w > MaxTermWords {
			errs = append(errs, fmt.Errorf("term %q has %d words, limit is %d", t, w, MaxTermWords))
		}
	}
	if realtime && total > MaxRealtimeChars {
		errs = append(errs, fmt.Errorf("terms total %d characters, limit is %d", total, MaxRealtimeChars))
	}
	return errors.Join(errs...)
}
