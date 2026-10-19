# ADR-0008: Answer contract — citations, "as of", refusal

**Status:** Accepted

## Decision
Every response from `/v1/ask`:
- has `status` ∈ `answered | partial | not_found`;
- includes `citations[]` (URL, title, doc type, academic year, section, document date, crawl date, quote) for everything stated, referenced from the text as `[n]`;
- includes `academic_year` (the year the answer applies to) and `as_of` (generation date) and `oldest_source_crawled_at` (the oldest crawl time among the cited sources);
- is `not_found` with a fixed-template message (in the user's language: "not found in official documents") when retrieval is below the relevance floor and there is no deadline row. No best-guess answers.
- reports `verified` (+ `warnings`) from post-generation checks: valid citation ids, and the **date guard**: every date in the text must be supported by a date in the provided context, compared as normalised calendar dates (not strings), so a Russian or English answer is checked against Italian sources.

Citations are streamed **before** the text.

**Source authority rule (`rag.SourceAuthorityRule`, ADR-0010):** the Italian original is cited first; an English translation is cited only after it and is flagged `informational_translation: true` (with `original_url`), and the answer text calls it "an informational translation". Citation order is not relevance order. The answer never presents the English text as legally binding, and may state that the regional resolution on which the call is based prevails in case of discrepancy without describing its content.

**Date format rule (in the answer system prompt, `rag.DateFormatRule`):** dates are always written with the month name in words, in any answer language (`30 settembre 2026`, `30 September 2026`, `30 сентября 2026`), never numerically. Numeric forms are ambiguous across locales (05/06/2026), a misread date can cost a student a scholarship, and a month name is read out unambiguously by text-to-speech. The guard stays **day-first** for numeric input (`datex`), but numeric dates in an answer are additionally reported as a `numeric_date` warning, since they break the rule.

## Consequences
- Some legitimate questions will get `not_found` (false refusals). Tracked in eval; the relevance floor is tuned on the golden set.
- The prompt and checks are language-independent: the guard works on normalised dates, not on the language of the answer.
