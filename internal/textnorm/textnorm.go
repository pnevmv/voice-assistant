// Package textnorm is the single text normalisation used for BOTH change
// detection (content_hash) and deadline quote verification. Using one function
// guarantees that "the quote is in the text" and "the text changed" mean the
// same thing, whatever the PDF extractor did to line breaks and typography.
package textnorm

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Rules, in order:
//  1. Unicode NFC.
//  2. Strip soft hyphens and zero-width characters; expand typographic ligatures.
//  3. Map apostrophe/quote/dash variants to ASCII ' " -.
//  4. Turn every Unicode space (NBSP, tabs, form feeds, newlines) into one space.
//  5. Join words split by a hyphen at a line break ("universi-\ntà" → "università").
//     Only when a letter precedes the hyphen and a LOWERCASE letter follows, so
//     "Art. 5-\nbis" and "socio-\nEconomico" keep their hyphen.
//  6. Collapse whitespace runs; trim.
//
// Case is preserved. Anything the extractor or an LLM changes beyond these
// rules makes a quote fail verification, which is the safe direction.
func Normalize(s string) string {
	s = norm.NFC.String(s)
	s = replacer.Replace(s)

	// Unify whitespace first so the hyphenation join sees a plain space.
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, s)
	s = hyphenBreak.ReplaceAllString(s, "$1$2")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

// ContentHash is the hex SHA-256 of Normalize(s); this is document_versions.content_hash.
func ContentHash(s string) string {
	sum := sha256.Sum256([]byte(Normalize(s)))
	return hex.EncodeToString(sum[:])
}

// QuoteInText reports whether quote occurs verbatim in text after both are
// normalised. An empty quote never matches.
func QuoteInText(text, quote string) bool {
	q := Normalize(quote)
	if q == "" {
		return false
	}
	return strings.Contains(Normalize(text), q)
}

// "letter - space(s) lowercase-letter". The regexp runs after whitespace was
// folded to single spaces, so a line break is just " ".
var hyphenBreak = regexp.MustCompile(`(\p{L})- +(\p{Ll})`)

// rep maps a code point to its replacement. Code points are written
// numerically so invisible characters stay visible in the source.
type rep struct {
	cp  rune
	to  string
	why string
}

var table = []rep{
	{0x00AD, "", "soft hyphen"},
	{0x200B, "", "zero width space"}, {0x200C, "", "zero width non-joiner"},
	{0x200D, "", "zero width joiner"}, {0x2060, "", "word joiner"}, {0xFEFF, "", "BOM / zero width no-break"},
	// ligatures (explicit list: NFKC would also rewrite ordinal indicators, superscripts, fractions)
	{0xFB00, "ff", "ligature"}, {0xFB01, "fi", "ligature"}, {0xFB02, "fl", "ligature"},
	{0xFB03, "ffi", "ligature"}, {0xFB04, "ffl", "ligature"}, {0xFB05, "st", "long s t"}, {0xFB06, "st", "ligature"},
	// apostrophes
	{0x2019, "'", "right single quote"}, {0x2018, "'", "left single quote"}, {0x201B, "'", "reversed single quote"},
	{0x02BC, "'", "modifier apostrophe"}, {0x00B4, "'", "acute accent"}, {0x0060, "'", "grave accent"}, {0x2032, "'", "prime"},
	// double quotes
	{0x201C, `"`, "left double quote"}, {0x201D, `"`, "right double quote"}, {0x201E, `"`, "low double quote"},
	{0x201F, `"`, "reversed double quote"}, {0x00AB, `"`, "guillemet left"}, {0x00BB, `"`, "guillemet right"}, {0x2033, `"`, "double prime"},
	// dashes and hyphens
	{0x2010, "-", "hyphen"}, {0x2011, "-", "non-breaking hyphen"}, {0x2012, "-", "figure dash"},
	{0x2013, "-", "en dash"}, {0x2014, "-", "em dash"}, {0x2212, "-", "minus sign"},
}

var replacer = func() *strings.Replacer {
	args := make([]string, 0, 2*len(table))
	for _, r := range table {
		args = append(args, string(r.cp), r.to)
	}
	return strings.NewReplacer(args...)
}()
