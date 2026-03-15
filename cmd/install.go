package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/usecases"
	"github.com/spf13/cobra"
)

func newInstallCommand(app *App) *cobra.Command {
	var (
		yes   bool
		force bool
	)

	cmd := &cobra.Command{
		Use:   "install",
		Args:  cobra.MinimumNArgs(1),
		Short: "install extensions by ids",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()

			installTargets, err := parseInstallTargets(args)
			if err != nil {
				return err
			}

			confirm := app.Presenter.ConfirmInstall
			if yes {
				confirm = func([]domain.ExtensionID, []domain.DownloadInfo, []domain.ReinstallInfo) bool { return true }
			}

			// Вызываем бизнес-логику
			ctx := cmd.Context()
			report, err := app.UseCase.Install(
				ctx,
				installTargets,
				usecases.InstallOpts{
					Confirm:           confirm,
					OnProgressFactory: app.Presenter.StartProgress,
					Force:             force,
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

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "force install extension")
	return cmd
}
