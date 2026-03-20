package ui

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
)

// Presenter - абстракция отображения (CLI, GUI, TUI...)
type Presenter interface {
	// StartProgress начинает прогресс-бар и возвращает ProgressFunc для обновления
	StartProgress(label string, total int64) (domain.ProgressFunc, func())

	// ShowMessage выводит информационное сообщение
	ShowMessage(msg string)

	// Log выводит логи
	Log(msg string, level domain.LogLevel)

	// Wait дождаться вывода всех сообщений
	Wait()

	// ShowExtensions выводит список установленных расширений
	ShowExtensions(extensions []domain.Extension)

	// ShowSearchResults выводит результаты поиска
	ShowSearchResults(extensions []domain.Extension)

	// ShowVersions выводит список версий расширения
	ShowVersions(versions []domain.VersionInfo)

	// ShowInstallResult выводит результаты установки расширений
	ShowInstallResult(res []domain.ExtensionResult)

	// ConfirmInstall подтверждает установку у пользователя
	ConfirmInstall(requestedIDs []domain.ExtensionID, extensions []domain.DownloadInfo, reinstall []domain.ReinstallInfo) bool

	// ShowRemoveResult выводит результаты удаления расширений
	ShowRemoveResult(res []domain.ExtensionResult)

	// ConfirmRemove подтверждает удаление у пользователя
	ConfirmRemove(requestedIDs []domain.ExtensionID, extensions []domain.Extension) bool

	// ShowUpdateResult выводит результаты обновления расширений
	ShowUpdateResult(res []domain.ExtensionResult)

	// ConfirmUpdate подтверждает обновление у пользователя
	ConfirmUpdate(toUpdate []domain.UpdateInfo) bool
}
