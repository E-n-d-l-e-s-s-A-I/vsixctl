package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/spf13/cobra"
)

func newInstallCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Args:  cobra.MinimumNArgs(1),
		Short: "install extensions by ids",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()

			// 1. Получаем все расширения с их зависимостями
			ids := make([]domain.ExtensionID, len(args))
			for i, id := range args {
				extensionID, err := domain.ParseExtensionID(id)
				if err != nil {
					return err
				}
				ids[i] = extensionID
			}
			ctx := cmd.Context()
			app.Presenter.ShowMessage("resolving dependencies...")
			resolved, alreadyInstalled, err := app.UseCase.InstallResolve(ctx, ids)
			if err != nil {
				return err
			}

			// 2. Выводим сообщения о уже установленных расширениях
			if len(alreadyInstalled) > 0 {
				alreadyInstalledErrors := make([]domain.ExtensionResult, len(alreadyInstalled))
				for i, id := range alreadyInstalled {
					alreadyInstalledErrors[i] = domain.ExtensionResult{
						ID:  id,
						Err: domain.ErrAlreadyInstalled,
					}
				}
				app.Presenter.ShowInstallResult(alreadyInstalledErrors)
			}
			if len(resolved) == 0 {
				return nil
			}

			// 3. Спрашиваем подтверждение у пользователя
			if ok := app.Presenter.ConfirmInstall(ids, resolved); !ok {
				return nil
			}

			// 4. Устанавливаем и выводим результат
			onProgressFactory := func(label string) (domain.ProgressFunc, func()) {
				return app.Presenter.StartProgress(label)
			}
			result := app.UseCase.Install(ctx, resolved, onProgressFactory)

			app.Presenter.Wait()
			// Добавляем пустую строку перед выводом
			app.Presenter.ShowMessage("")
			app.Presenter.ShowInstallResult(result)
			return nil
		},
	}
	return cmd
}
