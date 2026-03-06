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

			ids := make([]domain.ExtensionID, len(args))
			for i, id := range args {
				extensionID, err := domain.ParseExtensionID(id)
				if err != nil {
					return err
				}
				ids[i] = extensionID
			}
			onProgressFactory := func(label string) (domain.ProgressFunc, func()) {
				return app.Presenter.StartProgress(label)
			}
			results, err := app.UseCase.Install(cmd.Context(), ids, onProgressFactory)
			app.Presenter.Wait()
			if err != nil {
				return err
			}

			for _, res := range results {
				if res.Err != nil {
					app.Presenter.ShowError(res.Err)
				} else {
					app.Presenter.ShowMessage(res.ID.String() + " installed")
				}
			}
			return nil
		},
	}
	return cmd
}
