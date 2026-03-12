package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/usecases"
	"github.com/spf13/cobra"
)

func newInstallCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Args:  cobra.MinimumNArgs(1),
		Short: "install extensions by ids",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()

			ids, err := parseExtensionIDs(args)
			if err != nil {
				return err
			}

			// Вызываем бизнес-логику
			ctx := cmd.Context()
			report, err := app.UseCase.Install(
				ctx,
				ids,
				usecases.InstallOpts{
					Confirm:           app.Presenter.ConfirmInstall,
					OnProgressFactory: app.Presenter.StartProgress,
				},
			)
			if err != nil {
				return err
			}
			app.Presenter.Wait()

			// Выводим результат
			app.Presenter.ShowInstallResult(report.Results)
			return nil
		},
	}
	return cmd
}
