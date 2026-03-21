package logger

import (
	"testing"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

func TestLogger(t *testing.T) {
	type logEntry struct {
		msg   string
		level domain.LogLevel
	}

	tests := []struct {
		name     string
		logLevel domain.LogLevel
		call     func(l *Logger)
		want     []logEntry
	}{
		{
			name:     "debug_level_passes_all",
			logLevel: domain.LogDebug,
			call: func(l *Logger) {
				l.Debug("d")
				l.Info("i")
				l.Warn("w")
				l.Error("e")
			},
			want: []logEntry{
				{"d", domain.LogDebug},
				{"i", domain.LogInfo},
				{"w", domain.LogWarn},
				{"e", domain.LogError},
			},
		},
		{
			name:     "warn_level_filters_debug_and_info",
			logLevel: domain.LogWarn,
			call: func(l *Logger) {
				l.Debug("d")
				l.Info("i")
				l.Warn("w")
				l.Error("e")
			},
			want: []logEntry{
				{"w", domain.LogWarn},
				{"e", domain.LogError},
			},
		},
		{
			name:     "error_level_filters_all_except_error",
			logLevel: domain.LogError,
			call: func(l *Logger) {
				l.Debug("d")
				l.Info("i")
				l.Warn("w")
				l.Error("e")
			},
			want: []logEntry{
				{"e", domain.LogError},
			},
		},
		{
			name:     "info_level_filters_debug",
			logLevel: domain.LogInfo,
			call: func(l *Logger) {
				l.Debug("d")
				l.Info("i")
			},
			want: []logEntry{
				{"i", domain.LogInfo},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var got []logEntry
			logFunc := func(msg string, level domain.LogLevel) {
				got = append(got, logEntry{msg, level})
			}

			l := NewLogger(logFunc, testCase.logLevel)
			testCase.call(l)

			if len(got) != len(testCase.want) {
				t.Fatalf("got %d log entries, want %d", len(got), len(testCase.want))
			}
			for i := range got {
				if got[i] != testCase.want[i] {
					t.Errorf("entry %d: got %+v, want %+v", i, got[i], testCase.want[i])
				}
			}
		})
	}
}

func TestLoggerFormatArgs(t *testing.T) {
	var got string
	output := func(msg string, _ domain.LogLevel) {
		got = msg
	}

	l := NewLogger(output, domain.LogDebug)
	l.Warn("source %s unavailable: status %d", "https://example.com", 503)

	want := "source https://example.com unavailable: status 503"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLoggerNilFunc(t *testing.T) {
	l := NewLogger(nil, domain.LogDebug)
	// Не должен паниковать
	l.Debug("test")
	l.Info("test")
	l.Warn("test")
	l.Error("test")
}
