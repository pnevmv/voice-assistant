package speech

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidateKeyterms(t *testing.T) {
	many := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = "t" + strings.Repeat("x", i%5)
		}
		return out
	}
	tests := []struct {
		name     string
		terms    []string
		realtime bool
		wantErr  string // substring; "" = valid
	}{
		{"empty is valid", nil, true, ""},
		{"50 short terms realtime", many(50), true, ""},
		{"51 terms realtime", many(51), true, "limit is 50"},
		{"51 terms batch is fine", many(51), false, ""},
		{"1000 terms batch", many(1000), false, ""},
		{"1001 terms batch", many(1001), false, "limit is 1000"},
		{"term of exactly 50 chars", []string{strings.Repeat("a", 50)}, true, ""},
		{"term of 51 chars", []string{strings.Repeat("a", 51)}, true, "51 characters"},
		{"term of 51 chars batch", []string{strings.Repeat("a", 51)}, false, "51 characters"},
		{"five words ok", []string{"uno due tre quattro cinque"}, true, ""},
		{"six words", []string{"uno due tre quattro cinque sei"}, true, "6 words"},
		{"chars counted as runes not bytes", []string{strings.Repeat("è", 50)}, true, ""},
		{"total 1000 realtime ok", []string{strings.Repeat("a", 50), strings.Repeat("b", 50)}, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKeyterms(tt.terms, tt.realtime)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("error %v does not contain %q", err, tt.wantErr)
			}
		})
	}

	t.Run("total characters over the realtime limit", func(t *testing.T) {
		var terms []string
		for i := 0; i < 21; i++ { // 21 * 50 = 1050 chars, 21 terms
			terms = append(terms, strings.Repeat("a", 50))
		}
		if err := ValidateKeyterms(terms, true); err == nil || !strings.Contains(err.Error(), "1050 characters") {
			t.Fatalf("got %v", err)
		}
		if err := ValidateKeyterms(terms, false); err != nil {
			t.Fatalf("batch has no total limit: %v", err)
		}
	})

	t.Run("reports every violation", func(t *testing.T) {
		err := ValidateKeyterms([]string{strings.Repeat("a", 60), "a b c d e f"}, true)
		if err == nil || !strings.Contains(err.Error(), "60 characters") || !strings.Contains(err.Error(), "6 words") {
			t.Fatalf("got %v", err)
		}
	})
}

func TestLoadKeyterms(t *testing.T) {
	write := func(t *testing.T, body string) string {
		t.Helper()
		p := filepath.Join(t.TempDir(), "keyterms.txt")
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	t.Run("comments, blanks, whitespace and duplicates", func(t *testing.T) {
		got, err := LoadKeyterms(write(t, "# header\n\n  ISEE \nbando\n\nisee\nborsa di studio\r\n# tail\n"))
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"ISEE", "bando", "borsa di studio"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("rejects a list that would break on scribe_v3_live", func(t *testing.T) {
		var b strings.Builder
		for i := 0; i < 51; i++ {
			b.WriteString("term" + strings.Repeat("x", i) + "\n")
		}
		if _, err := LoadKeyterms(write(t, b.String())); err == nil || !strings.Contains(err.Error(), "limit is 50") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := LoadKeyterms(filepath.Join(t.TempDir(), "nope.txt")); err == nil {
			t.Fatal("expected error")
		}
	})
}

// The shipped list must always satisfy the realtime limits, so switching to
// scribe_v3_live is a configuration change only.
func TestShippedKeytermsFile(t *testing.T) {
	got, err := LoadKeyterms("../../config/keyterms.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ISEE", "ISEU", "CFU", "Uniweb", "Unipd", "bando", "borsa di studio",
		"immatricolazione", "ESU", "Ateneo", "graduatoria", "idoneo", "fuori sede"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
