package ui

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// Presenter - абстракция отображения (CLI, GUI, TUI...)
type Presenter interface {
	// ShowExtensions выводит список расширений
	ShowExtensions(extensions []domain.Extension)

	// ShowInstallResult выводит результаты установки расширений
	ShowInstallResult(res []domain.InstallResult)

	// StartProgress начинает прогресс-бар и возвращает ProgressFunc для обновления
	StartProgress(label string) (domain.ProgressFunc, func())

	// ShowMessage выводит информационное сообщение
	ShowMessage(msg string)

	// Подтверждаем установку у пользователя
	ConfirmInstall(requestedIDs []domain.ExtensionID, extensions map[domain.ExtensionID]domain.VersionInfo) bool

	// Log выводит логи
	Log(msg string)

	// Wait дождаться вывода всех сообщений
	Wait()
}
