package textnorm

import "testing"

// Invisible characters are spelled out so the test source stays readable.
var (
	softHyphen     = string(rune(0x00AD))
	zeroWidth      = string(rune(0x200B))
	nbsp           = string(rune(0x00A0))
	narrowNbsp     = string(rune(0x202F))
	combiningAcute = string(rune(0x0301))
)

// pdftotextLayout mimics `pdftotext -layout` on an Italian bando: hard-wrapped
// lines, hyphenation at line ends, two-column spacing, a form feed at the page
// break, curly apostrophes, a soft hyphen and an fi ligature.
var pdftotextLayout = "Art. 6 - Termini e modalità di presentazione della domanda\n" +
	"\n" +
	"La domanda di benefici deve essere presentata esclusivamente on-line\n" +
	"tramite il portale Uniweb entro le ore 12:00 del 30 set-\n" +
	"tembre 2026, a pena di esclu-\n" +
	"sione.   Per gli studenti dell’Uni" + softHyphen + "versità di Padova   non è richiesto\n" +
	"alcun ﬁle cartaceo.\n" +
	"\f" +
	"Art. 7 - Requisiti di merito"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"empty", "", ""},
		{"whitespace and newlines", "a \t b\n\n c\r\nd", "a b c d"},
		{"nbsp and narrow nbsp", "a" + nbsp + "b" + narrowNbsp + "c", "a b c"},
		{"hyphenated line break", "universi-\ntà", "università"},
		{"hyphen break with indent", "universi-\n    tà", "università"},
		{"hyphen before uppercase kept", "socio-\nEconomico", "socio- Economico"},
		{"hyphen after digit kept", "Art. 5-\nbis", "Art. 5- bis"},
		{"real hyphen same line kept", "on-line", "on-line"},
		{"apostrophes", "dell’Università dell‘ateneo dell`anno", "dell'Università dell'ateneo dell'anno"},
		{"double quotes", "«bando» “borsa” „x”", `"bando" "borsa" "x"`},
		{"dashes", "2026–27 — 2027‑28", "2026-27 - 2027-28"},
		{"soft hyphen and zero width", "Uni" + softHyphen + "ver" + zeroWidth + "sità", "Università"},
		{"ligatures", "ﬁle ﬂusso oﬃcio", "file flusso officio"},
		{"nfc: decomposed e-grave", "perche" + combiningAcute + "", "perché"},
		{"case preserved", "ISEE Uniweb", "ISEE Uniweb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.in); got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeIdempotent(t *testing.T) {
	for _, in := range []string{pdftotextLayout, "a  b\n c", "universi-\ntà", ""} {
		once := Normalize(in)
		if twice := Normalize(once); twice != once {
			t.Errorf("not idempotent: %q -> %q", once, twice)
		}
	}
}

func TestQuoteInText(t *testing.T) {
	tests := []struct {
		name, quote string
		want        bool
	}{
		{"quote re-flowed on one line", "entro le ore 12:00 del 30 settembre 2026, a pena di esclusione", true},
		{"quote keeps the hyphenation", "del 30 set-\ntembre 2026", true},
		{"quote with straight apostrophe", "studenti dell'Università di Padova", true},
		{"quote spanning the form feed", "alcun file cartaceo. Art. 7 - Requisiti di merito", true},
		{"quote with en dash instead of hyphen", "Art. 7 – Requisiti di merito", true},
		{"wrong date is not a match", "entro le ore 12:00 del 30 ottobre 2026", false},
		{"altered word is not a match", "entro le ore 12:00 del 30 settembre 2027", false},
		{"empty quote", "   ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := QuoteInText(pdftotextLayout, tt.quote); got != tt.want {
				t.Errorf("QuoteInText(%q) = %v, want %v", tt.quote, got, tt.want)
			}
		})
	}
}

// Cosmetic differences must not change the hash; real changes must.
func TestContentHash(t *testing.T) {
	base := "Entro il 30 settembre 2026 dell’Università."
	same := []string{
		"Entro  il 30 settembre\n2026 dell'Università.",
		"Entro il 30 set-\ntembre 2026 dell’Uni" + softHyphen + "versità.",
	}
	for _, s := range same {
		if ContentHash(s) != ContentHash(base) {
			t.Errorf("hash differs for cosmetic variant %q", s)
		}
	}
	if ContentHash("Entro il 30 ottobre 2026 dell’Università.") == ContentHash(base) {
		t.Error("hash equal for different date")
	}
	if len(ContentHash(base)) != 64 {
		t.Error("expected 64 hex chars")
	}
}
