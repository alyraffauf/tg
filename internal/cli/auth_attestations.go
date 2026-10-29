package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/internal/app"
	"github.com/spf13/cobra"
)

func newAttestationsCommand(service *app.Service) *cobra.Command {
	parent := &cobra.Command{Use: "attestations", Short: "Inspect and retry pending claims for saved knot changes"}
	parent.AddCommand(&cobra.Command{Use: "list", Short: "List pending attestations for the active account", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		result, err := service.ListPendingAttestations(cmd.Context())
		if err != nil {
			return err
		}
		return output(cmd, result, func(result *app.AttestationListResult) {
			for _, item := range result.Items {
				fmt.Fprintln(cmd.OutOrStdout(), item.Subject.URI)
			}
		})
	}})
	parent.AddCommand(&cobra.Command{Use: "retry", Short: "Retry pending attestations for the active account", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		result, err := service.RetryPendingAttestations(cmd.Context())
		if err != nil {
			return err
		}
		return output(cmd, result, func(result *app.AttestationRetryResult) {
			fmt.Fprintf(cmd.OutOrStdout(), "Attested %d changes; %d pending\n", result.Retried, result.Pending)
		})
	}})
	return parent
}
