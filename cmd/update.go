package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/usecases"
	"github.com/spf13/cobra"
)

func newUpdateCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "install extensions by ids",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()

			ids, err := parseExtensionIDs(args)
			if err != nil {
				return err
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
