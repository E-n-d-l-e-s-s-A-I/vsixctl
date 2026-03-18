package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/spf13/cobra"
)

func newVersionsCommand(app *App) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "versions <publisher.extension>",
		Args:  cobra.ExactArgs(1),
		Short: "List available versions of an extension",
		RunE: func(cmd *cobra.Command, args []string) error {
			defer app.Presenter.Wait()
			id, err := domain.ParseExtensionID(args[0])
			if err != nil {
				return err
			}
			versions, err := app.UseCase.Versions(cmd.Context(), id, limit)
			if err != nil {
				return err
			}
			app.Presenter.ShowVersions(versions)
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "results limit")
	return cmd
}
