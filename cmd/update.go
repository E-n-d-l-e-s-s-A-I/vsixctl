package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/spf13/cobra"
)

func newUpdateCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
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
			app.Presenter.ShowMessage("search for update...")
			resolved, notInstalled, err := app.UseCase.UpdateResolve(ctx, ids)
			if err != nil {
				return err
			}

			// 2. Выводим сообщения о уже установленных расширениях
			if len(notInstalled) > 0 {
				notInstalledErrors := make([]domain.ExtensionResult, len(notInstalled))
				for i, id := range notInstalled {
					notInstalledErrors[i] = domain.ExtensionResult{
						ID:  id,
						Err: domain.ErrAlreadyInstalled,
					}
				}
				// TODO поменять на showUpdateResult
				app.Presenter.ShowInstallResult(notInstalledErrors)
			}
			if len(resolved) == 0 {
				app.Presenter.ShowMessage("nothing to update")
				return nil
			}

			// 3. Спрашиваем подтверждение у пользователя
			ok := app.Presenter.ConfirmUpdate(resolved)
			if !ok {
				return nil
			}

			// 4. Устанавливаем и выводим результат
			// TODO
			return nil
		},
	}
	return cmd
}
