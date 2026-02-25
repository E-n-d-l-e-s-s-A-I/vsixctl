package cmd

import "github.com/spf13/cobra"

func newListCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Show installed extensions",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := app.UseCase.List(cmd.Context())
			if err != nil {
				return err
			}
			app.Presenter.ShowExtensions(results)
			return nil
		},
	}
	return cmd
}
