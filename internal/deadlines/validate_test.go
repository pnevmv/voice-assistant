package deadlines

import (
	"strings"
	"testing"
)

// Excerpts of the real 2026/27 bando as `pdftotext -layout` emits it: dates split
// across lines, curly apostrophes, uppercase headings, and the multi-column
// "RIEPILOGO SCADENZE" table whose columns are interleaved.
const bandoText = `3. COME CONCORRERE: MODALITA’ E SCADENZE DI PRESENTAZIONE DELLA DOMANDA
Per poter concorrere alla borsa di studio è necessario, ENTRO IL 30 SETTEMBRE 2026, compiere tutti i seguenti passaggi:

         della stessa. L’interessata/o riceve al proprio indirizzo di posta, personale o istituzionale una prima mail di
         ISEE. La RdA potrà essere modificata direttamente dall’utente, mediante la medesima procedura, entro il 30
         settembre 2026. La RdA salvata nella propria area personale di UNIWEB, costituisce unica prova di avvenuta

Per caricare la documentazione mancante la procedura online riaprirà dal 24 novembre 2026 al 30 aprile 2027 e

         precedenti carriere, entro il 10/08/2027.

                                                      Paga la 1° rata
  Agevolazioni su           Compila la DSU e
                                                      (anche se è di
  Uniweb entro il        richiedi l'ISEE entro il
                                                     16 euro) entro il
   30 settembre           30 settembre 2026
                                                    30 settembre 2026
        2026
`

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		c          Candidate
		wantQuote  bool
		wantDate   bool
		wantOK     bool
		wantInProb string
	}{
		{
			name: "single cut-off, quote re-flowed onto one line",
			c: Candidate{EndsOn: "2026-09-30",
				SourceQuote: "Per poter concorrere alla borsa di studio è necessario, ENTRO IL 30 SETTEMBRE 2026, compiere tutti i seguenti passaggi"},
			wantQuote: true, wantDate: true, wantOK: true,
		},
		{
			name: "date broken across a line in the source",
			c: Candidate{EndsOn: "2026-09-30",
				SourceQuote: "La RdA potrà essere modificata direttamente dall'utente, mediante la medesima procedura, entro il 30 settembre 2026."},
			wantQuote: true, wantDate: true, wantOK: true,
		},
		{
			name: "range dal … al …",
			c: Candidate{StartsOn: "2026-11-24", EndsOn: "2027-04-30",
				SourceQuote: "la procedura online riaprirà dal 24 novembre 2026 al 30 aprile 2027"},
			wantQuote: true, wantDate: true, wantOK: true,
		},
		{
			name:      "numeric date",
			c:         Candidate{EndsOn: "2027-08-10", SourceQuote: "precedenti carriere, entro il 10/08/2027."},
			wantQuote: true, wantDate: true, wantOK: true,
		},
		{
			name:      "hallucinated quote",
			c:         Candidate{EndsOn: "2026-09-30", SourceQuote: "Le domande vanno inviate entro il 30 settembre 2026 via PEC"},
			wantQuote: false, wantDate: true, wantOK: false, wantInProb: "source_quote",
		},
		{
			name:      "right quote, wrong date extracted",
			c:         Candidate{EndsOn: "2026-10-30", SourceQuote: "entro il 10/08/2027."},
			wantQuote: true, wantDate: false, wantOK: false, wantInProb: "2026-10-30",
		},
		{
			name:      "right quote, wrong year extracted",
			c:         Candidate{EndsOn: "2027-09-30", SourceQuote: "ENTRO IL 30 SETTEMBRE 2026, compiere tutti i seguenti passaggi"},
			wantQuote: true, wantDate: false, wantOK: false, wantInProb: "2027-09-30",
		},
		{
			name:      "range with only one end inside the quote",
			c:         Candidate{StartsOn: "2026-11-24", EndsOn: "2027-04-30", SourceQuote: "riaprirà dal 24 novembre 2026"},
			wantQuote: true, wantDate: false, wantOK: false, wantInProb: "2027-04-30",
		},
		{
			// The layout table interleaves columns, so no clean quote exists: the
			// validator must reject it rather than let a garbled quote through.
			name:      "interleaved summary-table quote is rejected",
			c:         Candidate{EndsOn: "2026-09-30", SourceQuote: "Agevolazioni su Uniweb entro il 30 settembre 2026"},
			wantQuote: false, wantDate: true, wantOK: false, wantInProb: "source_quote",
		},
		{
			name:      "no date at all",
			c:         Candidate{SourceQuote: "ENTRO IL 30 SETTEMBRE 2026"},
			wantQuote: true, wantDate: false, wantOK: false, wantInProb: "no date",
		},
		{
			name:      "malformed ISO",
			c:         Candidate{EndsOn: "30/09/2026", SourceQuote: "ENTRO IL 30 SETTEMBRE 2026"},
			wantQuote: true, wantDate: false, wantOK: false, wantInProb: "ISO",
		},
		{
			name:      "empty quote",
			c:         Candidate{EndsOn: "2026-09-30"},
			wantQuote: false, wantDate: false, wantOK: false, wantInProb: "source_quote",
		},
		{
			name:      "ends before starts",
			c:         Candidate{StartsOn: "2027-04-30", EndsOn: "2026-11-24", SourceQuote: "dal 24 novembre 2026 al 30 aprile 2027"},
			wantQuote: true, wantDate: true, wantOK: false, wantInProb: "before",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := Validate(tt.c, bandoText)
			if v.QuoteVerified != tt.wantQuote || v.DateInQuote != tt.wantDate || v.OK() != tt.wantOK {
				t.Fatalf("quote=%v date=%v ok=%v, want %v/%v/%v; problems: %v",
					v.QuoteVerified, v.DateInQuote, v.OK(), tt.wantQuote, tt.wantDate, tt.wantOK, v.Problems)
			}
			if tt.wantInProb != "" && !strings.Contains(strings.Join(v.Problems, "; "), tt.wantInProb) {
				t.Errorf("problems %v do not mention %q", v.Problems, tt.wantInProb)
			}
		})
	}
}
