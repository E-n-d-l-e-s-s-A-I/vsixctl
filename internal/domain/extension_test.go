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

func TestParseSearchType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    SearchType
		wantErr bool
	}{
		{
			name:  "text",
			input: "text",
			want:  SearchByText,
		},
		{
			name:  "id",
			input: "id",
			want:  SearchByID,
		},
		{
			name:  "name",
			input: "name",
			want:  SearchByName,
		},
		{
			name:    "invalid",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ParseSearchType(testCase.input)
			if (err != nil) != testCase.wantErr {
				t.Errorf("ParseSearchType(%q) error = %v, wantErr %v", testCase.input, err, testCase.wantErr)
				return
			}
			if !testCase.wantErr && got != testCase.want {
				t.Errorf("ParseSearchType(%q) = %v, want %v", testCase.input, got, testCase.want)
			}
		})
	}
}

func TestNewerThan(t *testing.T) {
	tests := []struct {
		name      string
		firstVer  Version
		secondVer Version
		want      bool
	}{
		{
			name:      "major newer",
			firstVer:  Version{1, 0, 0},
			secondVer: Version{0, 9, 11},
			want:      true,
		},
		{
			name:      "minor newer",
			firstVer:  Version{4, 2, 0},
			secondVer: Version{4, 1, 11},
			want:      true,
		},
		{
			name:      "patch newer",
			firstVer:  Version{1, 1, 11},
			secondVer: Version{1, 1, 10},
			want:      true,
		},
		{
			name:      "major older",
			firstVer:  Version{1, 9, 9},
			secondVer: Version{2, 0, 0},
			want:      false,
		},
		{
			name:      "minor older",
			firstVer:  Version{2, 1, 9},
			secondVer: Version{2, 2, 0},
			want:      false,
		},
		{
			name:      "patch older",
			firstVer:  Version{2, 2, 8},
			secondVer: Version{2, 2, 9},
			want:      false,
		},
		{
			name:      "equals",
			firstVer:  Version{1, 1, 1},
			secondVer: Version{1, 1, 1},
			want:      false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := testCase.firstVer.NewerThan(testCase.secondVer)
			if got != testCase.want {
				t.Errorf("got %v, want %v", got, testCase.want)
			}
		})
	}
}
