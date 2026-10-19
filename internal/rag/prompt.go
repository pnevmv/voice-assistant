package rag

// DateFormatRule is part of the answer system prompt (M3). Dates are always
// written with the month name, in any answer language. Numeric dates are
// ambiguous (05/06/2026 is 5 June in Italy and 6 May in the US), a misread date
// can cost a student a scholarship, and a month name is also what a
// text-to-speech voice reads out unambiguously. The date guard
// (internal/datex) still parses numeric forms, but only to catch answers that
// break this rule: those are flagged as warnings.
const DateFormatRule = "Write every date with the month spelled out in words in the language of your answer " +
	"(for example \"30 settembre 2026\", \"30 September 2026\", \"30 сентября 2026\"). " +
	"Never write a date numerically (not 30/09/2026, not 30.09.2026, not 2026-09-30), even if the source does. " +
	"Copy the day, month and year exactly from the provided sources."

// SourceAuthorityRule is the second authority-related part of the answer system
// prompt (ADR-0008, ADR-0010). The Italian bando is the legally valid text; the
// English version is an informational translation.
const SourceAuthorityRule = "Only the Italian version of a call (bando) is legally valid; the English version is an informational translation. " +
	"Cite the Italian original first. If you also cite an English translation, cite it after the original and say that it is an informational translation. " +
	"Never present the English text as legally binding. When asked which document prevails, say that in case of discrepancy the provisions of the regional resolution " +
	"(Deliberazione della Giunta regionale) on which the call is based prevail over the call, and do not describe the content of that resolution, which is not in your sources."
