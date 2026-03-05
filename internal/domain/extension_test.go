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
			want:    Version{Major: 1, Minor: 2, Patch: 0},
			wantErr: false,
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

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ParseVersion(testCase.input)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ParseVersion(%q) error = %v, wantErr %v", testCase.input, err, testCase.wantErr)
				return
			}
			if !testCase.wantErr && got != testCase.want {
				t.Errorf("ParseVersion(%q) = %v, want %v", testCase.input, got, testCase.want)
			}
		})
	}
}
