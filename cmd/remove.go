package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/usecases"
	"github.com/spf13/cobra"
)

func newRemoveCommand(app *App) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:     "remove <publisher.extension>...",
		Args:    cobra.MinimumNArgs(1),
		Short:   "remove extensions by ids",
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()

			ids, err := parseExtensionIDs(args)
			if err != nil {
				return err
			}

			confirm := app.Presenter.ConfirmRemove
			if yes {
				confirm = func([]domain.ExtensionID, []domain.Extension) bool { return true }
			}

			// Вызываем бизнес-логику
			ctx := cmd.Context()
			report, err := app.UseCase.Remove(
				ctx,
				ids,
				usecases.RemoveOpts{
					Confirm: confirm,
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
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}
