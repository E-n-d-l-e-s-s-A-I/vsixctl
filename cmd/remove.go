package cmd

import (
	"github.com/E-n-d-l-e-s-s-A-I/vsixctl/internal/domain"
	"github.com/spf13/cobra"
)

func newRemoveCommand(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Args:  cobra.MinimumNArgs(1),
		Short: "remove extensions by ids",
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
			ctx := cmd.Context()
			result := app.UseCase.Remove(ctx, ids)
			app.Presenter.ShowRemoveResult(result)
			return nil
		},
	}
	return cmd
}
