package ui

import "github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"

// Presenter — абстракция отображения (CLI, GUI, TUI...)
type Presenter interface {
	// ShowExtensions выводит список расширений
	ShowExtensions(extensions []domain.Extension)

	// StartProgress начинает прогресс-бар и возвращает ProgressFunc для обновления
	StartProgress(label string) (domain.ProgressFunc, FinishFunc)

	// ShowMessage выводит информационное сообщение
	ShowMessage(msg string)

	// ShowError выводит ошибку
	ShowError(err error)
}

type FinishFunc func()
