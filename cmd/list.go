package cmd

import (
	"github.com/spf13/cobra"
)

func newListCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()
			results, err := app.UseCase.List(cmd.Context())
			if err != nil {
				app.Presenter.ShowError(err)
				return err
			}
			app.Presenter.ShowExtensions(results)
			return nil
		},
		Use:           "list",
		Short:         "Show installed extensions",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	return cmd
}
