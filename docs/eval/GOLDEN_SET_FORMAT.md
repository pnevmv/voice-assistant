# Golden set format

The golden set is the ground truth for every metric. **Wrong or stale gold entries silently corrupt the eval**, so entries are verified by a human against the official source and carry the evidence for that verification.

- Files: `eval/golden/golden.jsonl` is the real set (first verified record: `auth-legal-001`); `eval/golden/examples.placeholder.jsonl` only illustrates the structure. One JSON object per line, UTF-8.
- Size target: 50–100 entries; coverage targets are in ROADMAP M4.
- Version: `eval/golden/VERSION` (semver-ish; bump on any change). The runner records the version and the SHA-256 of the files in `eval_runs`.
- **Every fact-bearing value in the shipped examples is the literal string `"TODO"`**: they illustrate structure only. An entry with `status: "placeholder"` is rejected by the validator unless `--allow-placeholders` is passed.

## Entry schema

```jsonc
{
  "id": "sch-deadline-001",             // unique, stable; prefix = topic
  "status": "verified",                 // placeholder | draft | verified | retired
  "category": "deadline",               // deadline | requirement | procedure | document_list | cross_year_trap | not_found | term_definition | source_authority
  "tags": ["scholarship", "borsa-regionale"],

  "questions": {                        // same question, up to three languages; at least one
    "it": "…",
    "en": "…",
    "ru": "…"
  },

  "academic_year": "2026/27",           // year the correct answer applies to; null only for not_found/evergreen
  "asked_year": null,                   // set when the question names a year explicitly (trap questions); null = default current year

  "expected": {
    "answer_type": "date",              // date | fact | list | not_found
    "reference_answer": {               // short, canonical, in Italian; used by RAGAS as the reference
      "it": "…"
    },
    "dates": [                          // for answer_type=date (or any answer containing dates)
      {
        "event_key": "scholarship_application_deadline",
        "starts_on": null,              // ISO 8601 or null
        "ends_on": "2026-00-00",        // ISO 8601, REQUIRED real date when verified
        "time_of_day": "12:00",         // optional, Europe/Rome
        "applies_to": "all"
      }
    ],
    "required_facts": [                 // atomic facts that must be present (for LLM/keyword check and context recall)
      "…"
    ],
    "forbidden": [                      // strings/dates that must NOT appear (wrong-year values, outdated amounts)
      "…"
    ],
    "first_citation_url": null,         // optional: the citation list must start with this URL (e.g. the Italian original)
    "informational_source_urls": [],    // optional: sources that must be marked as informational translations when cited
    "status": "answered"                // answered | not_found  (expected API status)
  },

  "sources": [                          // evidence for the gold answer; used for context-recall and citation checks
    {
      "url": "https://…",
      "doc_type": "bando",
      "section": "Art. 6",
      "quote": "verbatim text containing the fact",
      "document_date": "2026-07-10",    // date printed on the document
      "retrieved_on": "2026-11-24"      // when the human read it
    }
  ],

  "notes": "",                          // optional free text: caveats for reviewers and metric authors

  "time_sensitive": true,               // deadline/amount questions: answer may change when the bando is amended
  "valid_until": "2027-09-30",          // review again on/after this date; runner warns when expired

  "verification": {
    "verified_by": "initials",
    "verified_on": "2026-11-24",
    "method": "read bando PDF art. 6, compared with unipd.it page",
    "second_reviewer": null             // recommended for all date entries
  },

  "audio": {                            // optional, for voice eval
    "file": "audio/sch-deadline-001.ru.wav",
    "language": "ru",
    "transcript": "…",                  // exact words spoken (WER reference)
    "domain_terms": ["ISEE", "bando"],  // terms scored separately
    "recorded_by": "human | synthetic"
  }
}
```

## Field semantics and how they are scored

| Field | Used by metric |
|---|---|
| `questions.*` | Each language is run as a separate test case (`eval_results` unique on question id + language). |
| `expected.reference_answer` | RAGAS answer relevance / factual correctness reference. |
| `expected.dates` | **Date exact match**: the set of dates in the system's structured `deadlines[]` (and, secondarily, dates parsed from the answer text) must equal the expected set. Partial credit is not given: a date is right or wrong. |
| `academic_year`, `asked_year` | **Year correctness**: `answer.academic_year` equals the expected year **and** no citation belongs to another year (unless the question asked for it). |
| `expected.forbidden` | Hard fail if present in the answer (catches 2025/26 values leaking into 2026/27 answers). |
| `expected.first_citation_url`, `informational_source_urls` | **Citation order**: the first citation is the given URL (Italian original first), and every cited URL listed as informational carries `informational_translation: true` in the API response (ADR-0008, ADR-0010). |
| `expected.status` | Refusal precision/recall: `not_found` entries must not produce confident answers; answered entries must not be refused. |
| `sources[]` | **Context recall**: fraction of gold sources (URL + section) present in retrieved chunks; **citation correctness**: cited URLs ⊆ plausible sources. |
| `required_facts` | Faithfulness/recall cross-check and fact-level scoring. |
| `audio.*` | Voice: WER (overall + `domain_terms` term error rate), TTFA. Every audio entry is run **twice**, with and without STT keyterms (`keyterms=true|false`), and the term-level error is compared; TTFA is compared across TTS models (`tts_model`). Both overrides need `VOICE_ALLOW_OVERRIDES=true` on the voice service (off by default). |

## Categories — minimum coverage for M4

| Category | Min | Notes |
|---|---|---|
| `deadline` | 10 | each with `dates` and a second reviewer |
| `requirement` | 12 | e.g. eligibility conditions, thresholds (`required_facts`) |
| `procedure` | 8 | steps and where to apply |
| `document_list` | 6 | admission documents; `answer_type: list` |
| `cross_year_trap` | 5 | question names or implies the *other* year; `forbidden` lists the other year's values |
| `not_found` | 8 | plausible but unanswerable from official docs |
| `term_definition` | 3 | ISEE, CFU, Uniweb… |
| `source_authority` | 2 | which document/version is legally binding; citation order and "informational translation" marking |
| languages | ≥ 15 each of RU/EN | the same entry can carry all three languages |

## Authoring workflow

1. Read the official document (PDF/HTML), not the system's output. Never copy gold answers from the assistant.
2. Copy the *verbatim* quote and URL into `sources[]`; enter the date in ISO form from that quote.
3. Fill `verification`; set `status: "verified"` only after the steps above (second reviewer for dates).
4. Set `valid_until`. When a bando is amended (rettifica), re-verify affected entries and bump `eval/golden/VERSION`.
5. `retired` entries stay in the file for history and are skipped.

## Validator rules (implemented in M4)
Unique ids; required fields per `status`; `first_citation_url` / `informational_source_urls` must appear in `sources[]`; ISO dates; `academic_year` format; `verified` ⇒ non-empty `sources[].quote`, `verification.*`, and no `"TODO"` anywhere; `answer_type: date` ⇒ `dates` non-empty; `not_found` ⇒ no `dates`, `sources` may be empty; `valid_until` not in the past (warning).
