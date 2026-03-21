package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
}

// nopLogger - логгер-заглушка, который ничего не делает
type nopLogger struct{}

func (nopLogger) Debug(string, ...any) {}
func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Warn(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

// NopLogger возвращает логгер-заглушку
func NopLogger() Logger { return nopLogger{} }

type LogLevel int

const (
	LogDebug LogLevel = iota + 1
	LogInfo
	LogWarn
	LogError
)

func ParseLogLevel(s string) (LogLevel, error) {
	switch strings.ToLower(s) {
	case "debug":
		return LogDebug, nil
	case "info":
		return LogInfo, nil
	case "warn":
		return LogWarn, nil
	case "error":
		return LogError, nil
	}
	return 0, fmt.Errorf("parse log level: unknown log level: %s", s)
}

func (l LogLevel) String() string {
	switch l {
	case LogDebug:
		return "debug"
	case LogInfo:
		return "info"
	case LogWarn:
		return "warn"
	case LogError:
		return "error"
	default:
		return fmt.Sprintf("unknown(%d)", int(l))
	}
}

func (l LogLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

func (l *LogLevel) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	level, err := ParseLogLevel(s)
	if err != nil {
		return err
	}
	*l = level
	return nil
}
