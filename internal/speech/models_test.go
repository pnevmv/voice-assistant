package speech

import "testing"

func TestValidateSTTModel(t *testing.T) {
	tests := []struct {
		model   string
		wantErr bool
	}{
		{"scribe_v3", false},
		{"scribe_v3_live", false},
		{"scribe_v1", true},
		{"", true},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if err := ValidateSTTModel(tt.model); (err != nil) != tt.wantErr {
				t.Errorf("ValidateSTTModel(%q) error = %v, wantErr %v", tt.model, err, tt.wantErr)
			}
		})
	}
}
