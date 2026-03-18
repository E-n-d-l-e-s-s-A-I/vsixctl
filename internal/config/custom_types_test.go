package config

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDurationUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"seconds", `"2s"`, 2 * time.Second, false},
		{"milliseconds", `"500ms"`, 500 * time.Millisecond, false},
		{"minutes_and_seconds", `"1m30s"`, 90 * time.Second, false},
		{"invalid_string", `"abc"`, 0, true},
		{"number_instead_of_string", `123`, 0, true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(testCase.input), &d)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if time.Duration(d) != testCase.want {
				t.Errorf("got %v, want %v", time.Duration(d), testCase.want)
			}
		})
	}
}

func TestDurationMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input Duration
		want  string
	}{
		{"seconds", Duration(2 * time.Second), `"2s"`},
		{"milliseconds", Duration(500 * time.Millisecond), `"500ms"`},
		{"minutes_and_seconds", Duration(90 * time.Second), `"1m30s"`},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := json.Marshal(testCase.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != testCase.want {
				t.Errorf("got %s, want %s", got, testCase.want)
			}
		})
	}
}
