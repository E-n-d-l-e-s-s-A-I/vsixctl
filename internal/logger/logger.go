package logger

import (
	"fmt"

	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// OutputFunc - функция вывода лог-сообщения
type OutputFunc func(msg string, level domain.LogLevel)

type Logger struct {
	output   OutputFunc
	logLevel domain.LogLevel
}

func NewLogger(output OutputFunc, level domain.LogLevel) *Logger {
	if output == nil {
		output = func(string, domain.LogLevel) {}
	}
	return &Logger{output: output, logLevel: level}
}

func (l *Logger) Debug(format string, args ...any) {
	if l.logLevel <= domain.LogDebug {
		l.output(fmt.Sprintf(format, args...), domain.LogDebug)
	}
}

func (l *Logger) Info(format string, args ...any) {
	if l.logLevel <= domain.LogInfo {
		l.output(fmt.Sprintf(format, args...), domain.LogInfo)
	}
}

func (l *Logger) Warn(format string, args ...any) {
	if l.logLevel <= domain.LogWarn {
		l.output(fmt.Sprintf(format, args...), domain.LogWarn)
	}
}

func (l *Logger) Error(format string, args ...any) {
	if l.logLevel <= domain.LogError {
		l.output(fmt.Sprintf(format, args...), domain.LogError)
	}
}
