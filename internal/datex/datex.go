// Package datex finds calendar dates in Italian, English and Russian text and
// compares them as dates rather than as strings. It backs two safety checks:
//
//   - the deadline validator: is the extracted date really inside the quote?
//   - the answer date guard: does every date in the answer appear in the
//     retrieved context, even when the answer is Russian and the source Italian?
//
// Recognised forms: "30 settembre 2026", "30 September 2026", "September 30,
// 2026", "30 сентября 2026 г.", "30/09/2026", "30.09.2026", "2026-09-30",
// month-year ("marzo 2027") and ranges with shared parts ("dal 1 al 30
// settembre 2026"). Input is normalised with textnorm first, so line-broken and
// hyphenated PDF text is handled.
package datex

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"unipd-voice-assistant/internal/textnorm"
)

// Date may be partial: Year 0 = year not stated ("30 settembre"), Day 0 =
// month-level ("marzo 2027"). Partial dates act as wildcards in Supports.
type Date struct{ Year, Month, Day int }

func (d Date) Complete() bool { return d.Year > 0 && d.Month > 0 && d.Day > 0 }

// ISO renders YYYY-MM-DD, YYYY-MM for month-level and --MM-DD when the year is unknown.
func (d Date) ISO() string {
	switch {
	case d.Complete():
		return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
	case d.Day == 0:
		return fmt.Sprintf("%04d-%02d", d.Year, d.Month)
	default:
		return fmt.Sprintf("--%02d-%02d", d.Month, d.Day)
	}
}

// FromTime converts a time.Time to a complete Date (wall-clock fields).
func FromTime(t time.Time) Date { return Date{t.Year(), int(t.Month()), t.Day()} }

// Dates returns every date found in text, in order of appearance.
func Dates(text string) []Date {
	text = textnorm.Normalize(text)
	var found []hit
	taken := func(s, e int) bool {
		for _, h := range found {
			if s < h.end && h.start < e {
				return true
			}
		}
		return false
	}
	add := func(s, e int, ds ...Date) { found = append(found, hit{s, e, ds}) }

	// accept records a match; garbage (bad boundary, impossible month/day) is
	// still consumed so a later, looser pattern cannot re-read part of it.
	accept := func(s, e int, ok bool, ds ...Date) {
		if taken(s, e) {
			return
		}
		if !ok {
			ds = nil
		}
		add(s, e, ds...)
	}

	// 1. ranges with shared parts: "dal 1 al 30 settembre 2026"
	for _, m := range reRange.FindAllStringSubmatchIndex(text, -1) {
		g := groups(text, m)
		d1, d2 := atoi(g[1]), atoi(g[4])
		m1, m2 := monthNum(g[2]), monthNum(g[5])
		y1, y2 := atoi(g[3]), atoi(g[6])
		if m1 == 0 {
			m1 = m2
		}
		if y1 == 0 && y2 != 0 {
			y1 = y2
			if m1 > m2 { // "dal 15 dicembre al 10 gennaio 2027"
				y1--
			}
		}
		a, b := Date{y1, m1, d1}, Date{y2, m2, d2}
		accept(m[0], m[1], wordBoundary(text, m[0], m[1]) && plausible(a) && plausible(b), a, b)
	}
	// 2. ISO
	for _, m := range reISO.FindAllStringSubmatchIndex(text, -1) {
		g := groups(text, m)
		d := Date{atoi(g[1]), atoi(g[2]), atoi(g[3])}
		accept(m[0], m[1], plausible(d) && numericBoundary(text, m[0], m[1]), d)
	}
	// 3. numeric day-first: 30/09/2026, 30.09.2026, 30-09-26
	for _, m := range reNumeric.FindAllStringSubmatchIndex(text, -1) {
		g := groups(text, m)
		a, b, y := atoi(g[1]), atoi(g[2]), atoi(g[3])
		if len(g[3]) == 2 {
			y += 2000
		}
		d := Date{y, b, a}
		if b > 12 && a <= 12 { // "09/30/2026": only reading that is a real date
			d = Date{y, a, b}
		}
		accept(m[0], m[1], plausible(d) && numericBoundary(text, m[0], m[1]), d)
	}
	// 4. day month [year]: "30 settembre 2026", "30 September 2026", "30 сентября 2026"
	for _, m := range reDayMonth.FindAllStringSubmatchIndex(text, -1) {
		g := groups(text, m)
		d := Date{atoi(g[3]), monthNum(g[2]), atoi(g[1])}
		accept(m[0], m[1], plausible(d) && wordBoundary(text, m[0], m[1]), d)
	}
	// 5. English month day[, year]: "September 30, 2026"
	for _, m := range reMonthDay.FindAllStringSubmatchIndex(text, -1) {
		if m[1] < len(text) && text[m[1]] >= '0' && text[m[1]] <= '9' {
			continue // "March 2027" read as "March 20" + "27": leave it to the month-year pattern
		}
		g := groups(text, m)
		d := Date{atoi(g[3]), monthNum(g[1]), atoi(g[2])}
		modal := strings.EqualFold(g[1], "may") && d.Year == 0 // "students may 30 days…"
		accept(m[0], m[1], !modal && plausible(d) && wordBoundary(text, m[0], m[1]), d)
	}
	// 6. month year: "marzo 2027", "March 2027", "сентябрь 2026"
	for _, m := range reMonthYear.FindAllStringSubmatchIndex(text, -1) {
		g := groups(text, m)
		d := Date{atoi(g[2]), monthNum(g[1]), 0}
		accept(m[0], m[1], d.Month > 0 && wordBoundary(text, m[0], m[1]), d)
	}

	sort.Slice(found, func(i, j int) bool { return found[i].start < found[j].start })
	var out []Date
	for _, h := range found {
		out = append(out, h.dates...)
	}
	return out
}

// Contains reports whether text contains exactly the complete date t (year, month, day).
func Contains(text string, t time.Time) bool {
	want := FromTime(t)
	for _, d := range Dates(text) {
		if d == want {
			return true
		}
	}
	return false
}

// Supports reports whether context justifies the date d from an answer.
// Unknown parts of a context date (no year) are wildcards; an answer date that
// names a day needs a context date with the same day and month.
func Supports(context []Date, d Date) bool {
	for _, c := range context {
		if d.Day > 0 && (c.Day != d.Day || c.Month != d.Month) {
			continue
		}
		if d.Day == 0 && c.Month != d.Month {
			continue
		}
		if d.Year != 0 && c.Year != 0 && d.Year != c.Year {
			continue
		}
		return true
	}
	return false
}

// Unsupported returns the dates in answer that nothing in context supports.
// An empty result means the date guard passes.
func Unsupported(answer, context string) []Date {
	ctx := Dates(context)
	var bad []Date
	for _, d := range Dates(answer) {
		if !Supports(ctx, d) {
			bad = append(bad, d)
		}
	}
	return bad
}

// ---- internals ----

type hit struct {
	start, end int
	dates      []Date
}

// months must be a package-level var (not filled in init) because the regexps
// below are built from it, and var initialisation runs before init().
var months = buildMonths()

func buildMonths() map[string]int {
	months := map[string]int{}
	add := func(n int, names ...string) {
		for _, s := range names {
			months[s] = n
		}
	}
	add(1, "gennaio", "gen", "january", "jan", "января", "январь")
	add(2, "febbraio", "feb", "february", "февраля", "февраль")
	add(3, "marzo", "mar", "march", "марта", "март")
	add(4, "aprile", "apr", "april", "апреля", "апрель")
	add(5, "maggio", "mag", "may", "мая", "май")
	add(6, "giugno", "giu", "june", "jun", "июня", "июнь")
	add(7, "luglio", "lug", "july", "jul", "июля", "июль")
	add(8, "agosto", "ago", "august", "aug", "августа", "август")
	add(9, "settembre", "set", "september", "sept", "sep", "сентября", "сентябрь")
	add(10, "ottobre", "ott", "october", "oct", "октября", "октябрь")
	add(11, "novembre", "nov", "november", "ноября", "ноябрь")
	add(12, "dicembre", "dic", "december", "dec", "декабря", "декабрь")
	return months
}

func monthNum(s string) int {
	return months[strings.ToLower(strings.TrimSuffix(s, "."))]
}

func monthAlt(filter func(string) bool) string {
	var names []string
	for n := range months {
		if filter == nil || filter(n) {
			names = append(names, regexp.QuoteMeta(n))
		}
	}
	// longest first: RE2 alternation is leftmost-first, "marzo" must beat "mar"
	sort.Slice(names, func(i, j int) bool {
		li, lj := utf8.RuneCountInString(names[i]), utf8.RuneCountInString(names[j])
		if li != lj {
			return li > lj
		}
		return names[i] < names[j]
	})
	return strings.Join(names, "|")
}

func isEnglish(n string) bool {
	for _, r := range n {
		if r > unicode.MaxASCII {
			return false
		}
	}
	switch n { // Italian-only abbreviations that are ordinary English words
	case "set", "mag", "giu", "lug", "ago", "ott", "dic", "gen", "gennaio", "febbraio", "marzo", "aprile",
		"maggio", "giugno", "luglio", "agosto", "settembre", "ottobre", "novembre", "dicembre":
		return false
	}
	return true
}

const (
	dayRe  = `(\d{1,2})(?:[\x{00BA}\x{00B0}]|st|nd|rd|th)?`
	yearRe = `(?:\s*,?\s*((?:19|20)\d{2}))?`
)

var (
	anyMonth = `(` + monthAlt(nil) + `)\.?`
	enMonth  = `(` + monthAlt(isEnglish) + `)\.?`

	reISO       = regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
	reNumeric   = regexp.MustCompile(`(\d{1,2})[/.\-](\d{1,2})[/.\-](\d{4}|\d{2})`)
	reDayMonth  = regexp.MustCompile(`(?i)` + dayRe + `\s+(?:of\s+)?` + anyMonth + yearRe)
	reMonthDay  = regexp.MustCompile(`(?i)` + enMonth + `\s+` + dayRe + yearRe)
	reMonthYear = regexp.MustCompile(`(?i)` + anyMonth + `\s*,?\s*((?:19|20)\d{2})`)
	// groups: 1 day1, 2 month1?, 3 year1?, 4 day2, 5 month2, 6 year2?
	reRange = regexp.MustCompile(`(?i)(?:dal|dall'|from|с|tra\s+il|between)\s+` + dayRe +
		`(?:\s+` + anyMonth + `)?` + yearRe +
		`\s+(?:al|fino\s+al|to|until|through|по|до|e\s+il|and)\s+` + dayRe + `\s+` + anyMonth + yearRe)
)

// groups returns the submatch strings for a FindAllStringSubmatchIndex result
// ("" for groups that did not participate).
func groups(text string, idx []int) []string {
	g := make([]string, len(idx)/2)
	for i := range g {
		if idx[2*i] >= 0 {
			g[i] = text[idx[2*i]:idx[2*i+1]]
		}
	}
	return g
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

// plausible checks month and day ranges only. Calendar-impossible dates such
// as "31 settembre" are deliberately kept: nothing in a source can support them,
// so the date guard flags them, instead of the text silently degrading to
// "September".
func plausible(d Date) bool {
	return d.Month >= 1 && d.Month <= 12 && d.Day >= 1 && d.Day <= 31
}

func isWord(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// wordBoundary rejects matches glued to neighbouring letters or digits
// ("130 settembre", "30 settembrino"). RE2's \b is ASCII-only, so it is done by hand.
func wordBoundary(text string, start, end int) bool {
	if start > 0 {
		if r, _ := utf8.DecodeLastRuneInString(text[:start]); isWord(r) {
			return false
		}
	}
	if end < len(text) {
		if r, _ := utf8.DecodeRuneInString(text[end:]); isWord(r) {
			return false
		}
	}
	return true
}

// numericBoundary additionally rejects a match that is a slice of a longer
// number chain such as "1.30.09.2026".
func numericBoundary(text string, start, end int) bool {
	if !wordBoundary(text, start, end) {
		return false
	}
	if start >= 2 && strings.ContainsRune("/.-", rune(text[start-1])) && text[start-2] >= '0' && text[start-2] <= '9' {
		return false
	}
	if end+1 < len(text) && strings.ContainsRune("/.-", rune(text[end])) && text[end+1] >= '0' && text[end+1] <= '9' {
		return false
	}
	return true
}
