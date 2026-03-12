package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/usecases"
	"github.com/spf13/cobra"
)

func newUpdateCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "install extensions by ids",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()

			// Парсим id
			ids := make([]domain.ExtensionID, len(args))
			for i, id := range args {
				extensionID, err := domain.ParseExtensionID(id)
				if err != nil {
					return err
				}
				ids[i] = extensionID
			}

			// Вызываем бизнес-логику
			ctx := cmd.Context()
			report, err := app.UseCase.Update(
				ctx,
				ids,
				usecases.UpdateOpts{
					Confirm:           app.Presenter.ConfirmUpdate,
					OnProgressFactory: app.Presenter.StartProgress,
				},
			)
			if err != nil {
				return err
			}
			app.Presenter.Wait()

			// Выводим результат
			app.Presenter.ShowUpdateResult(report.Results)
			return nil
		},
	}
	return cmd
}
