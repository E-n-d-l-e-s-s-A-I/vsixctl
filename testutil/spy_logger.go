package testutil

import "fmt"

// SpyLogger собирает лог-сообщения для проверки в тестах
type SpyLogger struct {
	Debugs []string
	Infos  []string
	Warns  []string
	Errors []string
}

func (s *SpyLogger) Debug(format string, args ...any) {
	s.Debugs = append(s.Debugs, fmt.Sprintf(format, args...))
}

func (s *SpyLogger) Info(format string, args ...any) {
	s.Infos = append(s.Infos, fmt.Sprintf(format, args...))
}

func (s *SpyLogger) Warn(format string, args ...any) {
	s.Warns = append(s.Warns, fmt.Sprintf(format, args...))
}

func (s *SpyLogger) Error(format string, args ...any) {
	s.Errors = append(s.Errors, fmt.Sprintf(format, args...))
}
