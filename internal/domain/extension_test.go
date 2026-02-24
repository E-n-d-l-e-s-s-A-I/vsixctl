package domain

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Version
		wantErr bool
	}{
		{
			name:    "valid_version",
			input:   "1.2.3",
			want:    Version{Major: 1, Minor: 2, Patch: 3},
			wantErr: false,
		},
		{
			name:    "zero_version",
			input:   "0.0.0",
			want:    Version{Major: 0, Minor: 0, Patch: 0},
			wantErr: false,
		},
		{
			name:    "large_numbers",
			input:   "100.200.300",
			want:    Version{Major: 100, Minor: 200, Patch: 300},
			wantErr: false,
		},
		{
			name:    "non_numeric",
			input:   "a.b.c",
			wantErr: true,
		},
		{
			name:    "too_few_parts",
			input:   "1.2",
			wantErr: true,
		},
		{
			name:    "too_many_parts",
			input:   "1.2.3.4",
			wantErr: true,
		},
		{
			name:    "empty_string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseVersion(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
