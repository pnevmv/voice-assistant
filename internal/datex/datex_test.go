package datex

import (
	"reflect"
	"testing"
	"time"
)

func isos(ds []Date) []string {
	var out []string
	for _, d := range ds {
		out = append(out, d.ISO())
	}
	return out
}

func TestDates(t *testing.T) {
	tests := []struct {
		name, in string
		want     []string
	}{
		// Italian
		{"it long form", "30 settembre 2026", []string{"2026-09-30"}},
		{"it with time prefix", "entro le ore 12:00 del 30 settembre 2026", []string{"2026-09-30"}},
		{"it uppercase as in the bando", "ENTRO IL 30 SETTEMBRE 2026, compiere", []string{"2026-09-30"}},
		{"it ordinal first", "dal 1° settembre 2026", []string{"2026-09-01"}},
		{"it slash", "entro il 10/08/2027", []string{"2027-08-10"}},
		{"it dots", "scade il 30.09.2026.", []string{"2026-09-30"}},
		{"it dashes two-digit year", "30-09-26", []string{"2026-09-30"}},
		{"it abbreviation", "30 set. 2026", []string{"2026-09-30"}},
		{"it line broken as pdftotext emits it", "entro il 30\n         settembre 2026. La RdA", []string{"2026-09-30"}},
		{"it hyphenated across lines", "entro il 30 set-\ntembre 2026", []string{"2026-09-30"}},
		{"it year-less", "entro il 15 ottobre", []string{"--10-15"}},
		{"it month-year", "entro marzo 2027", []string{"2027-03"}},
		{"iso", "valid until 2026-09-30.", []string{"2026-09-30"}},
		{"two dates", "il 15 ottobre 2026 e il 25 novembre 2026", []string{"2026-10-15", "2026-11-25"}},
		// ranges
		{"range both complete (real bando)", "riaprirà dal 24 novembre 2026 al 30 aprile 2027 e", []string{"2026-11-24", "2027-04-30"}},
		{"range shared month and year", "dal 1 al 30 settembre 2026", []string{"2026-09-01", "2026-09-30"}},
		{"range shared year", "dal 15 luglio al 30 settembre 2026", []string{"2026-07-15", "2026-09-30"}},
		{"range across new year", "dal 15 dicembre al 10 gennaio 2027", []string{"2026-12-15", "2027-01-10"}},
		{"range tra il", "tra il 1 e il 5 ottobre 2026", []string{"2026-10-01", "2026-10-05"}},
		{"en range", "from 1 to 30 September 2026", []string{"2026-09-01", "2026-09-30"}},
		{"ru range", "с 1 по 30 сентября 2026 г.", []string{"2026-09-01", "2026-09-30"}},
		// English
		{"en day month", "30 September 2026", []string{"2026-09-30"}},
		{"en month first", "September 30, 2026", []string{"2026-09-30"}},
		{"en ordinal", "30th of September 2026", []string{"2026-09-30"}},
		{"en abbreviation", "Sept 30, 2026", []string{"2026-09-30"}},
		{"en month-year not day", "by March 2027", []string{"2027-03"}},
		{"en us numeric when day>12", "09/30/2026", []string{"2026-09-30"}},
		{"en may modal verb", "students may 30 days later", nil},
		{"en may with year", "May 5, 2027", []string{"2027-05-05"}},
		// Russian
		{"ru genitive", "30 сентября 2026", []string{"2026-09-30"}},
		{"ru with year suffix", "до 30 сентября 2026 г. включительно", []string{"2026-09-30"}},
		{"ru uppercase", "30 СЕНТЯБРЯ 2026", []string{"2026-09-30"}},
		{"ru month-year", "в марте... март 2027", []string{"2027-03"}},
		{"ru numeric", "до 30.09.2026", []string{"2026-09-30"}},
		// rejects
		// calendar-impossible dates are kept so that the guard can flag them
		{"impossible date kept", "31 settembre 2026", []string{"2026-09-31"}},
		{"leap day invalid kept", "29 febbraio 2027", []string{"2027-02-29"}},
		{"leap day valid", "29 febbraio 2028", []string{"2028-02-29"}},
		{"glued digits", "130 settembre 2026", nil},
		{"version-like chain", "v1.30.09.2026x", nil},
		{"time is not a date", "ore 12:30", nil},
		{"euro amount", "16 euro entro il", nil},
		{"empty", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isos(Dates(tt.in))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Dates(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	d := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name, text string
		want       bool
	}{
		{"plain", "entro il 30 settembre 2026", true},
		{"with hour", "entro le ore 12:00 del 30 settembre 2026", true},
		{"numeric", "scadenza 30/09/2026", true},
		{"dots", "scadenza 30.09.2026", true},
		{"end of a range", "dal 1 al 30 settembre 2026", true},
		{"other day", "entro il 29 settembre 2026", false},
		{"other year", "entro il 30 settembre 2027", false},
		{"year-less is not enough", "entro il 30 settembre", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(tt.text, d); got != tt.want {
				t.Errorf("Contains(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

// The context is Italian (as the sources are); answers come in three languages.
func TestUnsupported(t *testing.T) {
	const ctx = "La domanda va presentata ENTRO IL 30 SETTEMBRE 2026 alle ore 12:00. " +
		"Le graduatorie provvisorie saranno pubblicate il 15 ottobre 2026 e quelle definitive il 25/11/2026. " +
		"La procedura riaprirà dal 24 novembre 2026 al 30 aprile 2027; scorrimenti entro marzo 2027."
	tests := []struct {
		name, answer string
		want         []string // unsupported dates
	}{
		{"it ok", "Entro il 30 settembre 2026 [1].", nil},
		{"en ok", "The deadline is September 30, 2026 at noon [1].", nil},
		{"en ok day-first", "Apply by 30 September 2026.", nil},
		{"ru ok", "Подать заявку нужно до 30 сентября 2026 года [1].", nil},
		{"ru ok numeric", "Окончательные списки — 25.11.2026.", nil},
		{"ru ok range", "Загрузка документов: с 24 ноября 2026 по 30 апреля 2027.", nil},
		{"en ok month-level", "Further rankings by March 2027.", nil},
		{"ru ok month-level", "Дополнительные списки — март 2027.", nil},
		{"year-less answer date is supported by any year", "Provisional rankings on 15 October.", nil},
		{"it wrong day", "Entro il 29 settembre 2026.", []string{"2026-09-29"}},
		{"it wrong year", "Entro il 30 settembre 2027.", []string{"2027-09-30"}},
		{"en wrong month", "The deadline is October 30, 2026.", []string{"2026-10-30"}},
		{"ru wrong day", "Срок подачи — 1 октября 2026.", []string{"2026-10-01"}},
		{"ru wrong year numeric", "Окончательные списки — 25.11.2025.", []string{"2025-11-25"}},
		{"one right, one wrong", "Provisional 15 October 2026, final 26 November 2026.", []string{"2026-11-26"}},
		{"month-level supported by a day in that month", "Rankings by April 2027.", nil},
		{"month-level wrong", "Rankings by June 2027.", []string{"2027-06"}},
		{"impossible date is flagged", "Entro il 31 settembre 2026.", []string{"2026-09-31"}},
		{"no dates", "Il termine è riportato nel bando.", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isos(Unsupported(tt.answer, ctx))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unsupported(%q) = %v, want %v", tt.answer, got, tt.want)
			}
		})
	}
}

// Structured deadline rows reach the prompt as ISO dates; answers in any
// language must still be supported by them.
func TestUnsupportedAgainstISOContext(t *testing.T) {
	const ctx = "ends_on: 2026-09-30\nstarts_on: 2026-11-24"
	if got := Unsupported("до 30 сентября 2026", ctx); len(got) != 0 {
		t.Errorf("unexpected unsupported: %v", got)
	}
	if got := Unsupported("by October 1, 2026", ctx); len(got) != 1 {
		t.Errorf("expected one unsupported, got %v", got)
	}
}
