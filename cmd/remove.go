package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/spf13/cobra"
)

func newRemoveCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Args:  cobra.MinimumNArgs(1),
		Short: "remove extensions by ids",
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
			resolved, notInstalled, err := app.UseCase.RemoveResolve(ctx, ids)
			if err != nil {
				return err
			}

			// 2. Выводим сообщения о неустановленных расширениях
			if len(notInstalled) != 0 {
				notInstalledErrors := make([]domain.ExtensionResult, len(notInstalled))
				for i, id := range notInstalled {
					notInstalledErrors[i] = domain.ExtensionResult{
						ID:  id,
						Err: domain.ErrNotInstalled,
					}
				}
				app.Presenter.ShowRemoveResult(notInstalledErrors)
			}
			if len(resolved) == 0 {
				return nil
			}

			// 3. Спрашиваем подтверждение у пользователя
			if ok := app.Presenter.ConfirmRemove(ids, resolved); !ok {
				return nil
			}

			// 4. Удаляем и выводим результат
			toDelete := make([]domain.ExtensionID, len(resolved))
			for i, ext := range resolved {
				toDelete[i] = ext.ID
			}
			result := app.UseCase.Remove(ctx, toDelete)
			app.Presenter.ShowMessage("")
			app.Presenter.ShowRemoveResult(result)
			return nil
		},
	}
	return cmd
}
