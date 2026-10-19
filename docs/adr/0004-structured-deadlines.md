# ADR-0004: Structured deadlines table with verified quotes

**Status:** Accepted

## Context
Vector search over prose is unreliable for "what is the deadline for X": it may retrieve the wrong year, a related-but-different deadline, or return a paragraph where the date is one of many. Wrong dates are the highest-cost error.

## Decision
An indexer step extracts records `{event, event_key, applies_to, academic_year, starts_on/ends_on, time_of_day, date_text, source_url, source_quote}` from each bando/avviso version using the gateway's `extract` alias with a JSON schema. Records are **validated before they are served**:
1. `source_quote` must be a substring of the version's text **after the shared normalisation** (`internal/textnorm`: NFC, whitespace/line-break collapse, hyphenated line-break join, quote/apostrophe/dash unification, ligature and soft-hyphen stripping), the same function that produces `content_hash`;
2. every claimed date must appear as a complete calendar date inside the quote, parsed from Italian/English/numeric forms and ranges (`internal/datex`);
3. `academic_year` must match the version's year;
4. `event_key` must come from a controlled vocabulary (extendable, in code).

Only validated, non-rejected rows are served, through a single database view, `servable_deadlines` (`is_current AND quote_verified AND date_in_quote AND review_status <> 'rejected'`); no query reads `deadlines` directly for serving. Deadlines are extracted from the Italian original of a bando only (ADR-0010). A human review state (`unreviewed → confirmed | rejected`) is tracked; the API exposes `reviewed`. Rows follow the version lifecycle (`is_current`).

Deadline-intent questions read this table, and the answer prompt receives rows with quotes. A date guard checks the generated text against the context.

## Consequences
- Adds an extraction pass (cost per new bando version: a few LLM calls) and a small review workflow.
- Dates are exact-matchable in eval (`date_exact_match`) independent of phrasing.
- Extraction errors are contained: bad rows are hidden, and the fallback is prose retrieval with a "verify at source" note.
