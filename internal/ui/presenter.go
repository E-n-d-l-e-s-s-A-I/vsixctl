package ui

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// Presenter - абстракция отображения (CLI, GUI, TUI...)
type Presenter interface {
	// ShowExtensions выводит список расширений
	ShowExtensions(extensions []domain.Extension)

	// ShowInstallResult выводит результаты установки расширений
	ShowInstallResult(res []domain.ExtensionResult)

	// ShowRemoveResult выводит результаты удаления расширений
	ShowRemoveResult(res []domain.ExtensionResult)

	// ShowUpdateResult выводит результаты обновления расширений
	ShowUpdateResult(res []domain.ExtensionResult)

	// ShowError форматирует и выводит ошибку
	ShowError(err error)

	// StartProgress начинает прогресс-бар и возвращает ProgressFunc для обновления
	StartProgress(label string) (domain.ProgressFunc, func())

	// ShowMessage выводит информационное сообщение
	ShowMessage(msg string)

	// ConfirmInstall подтверждает установку у пользователя
	ConfirmInstall(requestedIDs []domain.ExtensionID, extensions []domain.DownloadInfo) bool

	// ConfirmRemove подтверждает удаление у пользователя
	ConfirmRemove(requestedIDs []domain.ExtensionID, extensions []domain.Extension) bool

	// ConfirmUpdate подтверждает обновление у пользователя
	ConfirmUpdate(toUpdate []domain.UpdateInfo) bool

	// Log выводит логи
	Log(msg string)

	// Wait дождаться вывода всех сообщений
	Wait()
}
