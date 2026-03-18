package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/usecases"
	"github.com/spf13/cobra"
)

func newUpdateCommand(app *App) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "update [publisher.extension]...",
		Short: "update installed extensions",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()

			ids, err := parseExtensionIDs(args)
			if err != nil {
				return err
			}

			confirm := app.Presenter.ConfirmUpdate
			if yes {
				confirm = func([]domain.UpdateInfo) bool { return true }
			}

			// Вызываем бизнес-логику
			ctx := cmd.Context()
			report, err := app.UseCase.Update(
				ctx,
				ids,
				usecases.UpdateOpts{
					Confirm:           confirm,
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

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
