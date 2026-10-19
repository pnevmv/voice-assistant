package platform

import "testing"

func TestBool(t *testing.T) {
	tests := []struct {
		name, value string
		set         bool
		def, want   bool
	}{
		{"unset uses default false", "", false, false, false},
		{"unset uses default true", "", false, true, true},
		{"empty uses default", "", true, false, false},
		{"true", "true", true, false, true},
		{"1", "1", true, false, true},
		{"false overrides default true", "false", true, true, false},
		{"typo never enables", "ture", true, false, false},
		{"yes is not a ParseBool form", "yes", true, false, false},
		{"t", "t", true, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv("TEST_BOOL", tt.value)
			}
			if got := Bool("TEST_BOOL", tt.def); got != tt.want {
				t.Errorf("Bool(%q, %v) = %v, want %v", tt.value, tt.def, got, tt.want)
			}
		})
	}
}
