package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/usecases"
	"github.com/spf13/cobra"
)

func newRemoveCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Args:  cobra.MinimumNArgs(1),
		Short: "remove extensions by ids",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()

			ids, err := parseExtensionIDs(args)
			if err != nil {
				return err
			}

			// Вызываем бизнес-логику
			ctx := cmd.Context()
			report, err := app.UseCase.Remove(
				ctx,
				ids,
				usecases.RemoveOpts{
					Confirm: app.Presenter.ConfirmRemove,
				},
			)
			if err != nil {
				return err
			}
			app.Presenter.Wait()

			// Выводим результат
			app.Presenter.ShowRemoveResult(report.Results)
			return nil
		},
	}
	return cmd
}
