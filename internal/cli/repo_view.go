package cli

import (
	"fmt"

	"github.com/alyraffauf/tg/tangled"
	"github.com/spf13/cobra"
)

var repoViewCmd = &cobra.Command{
	Use:   "view <handle/repo>",
	Short: "View a Tangled repository",
	Long:  `View details for a Tangled repository.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		handle, repo, err := parseHandleRepo(args[0])
		if err != nil {
			return err
		}

		ident, err := resolver.ResolveHandle(ctx, handle)
		if err != nil {
			return fmt.Errorf("resolve handle %q: %w", handle, err)
		}

		repoURI := fmt.Sprintf("at://%s/sh.tangled.repo/%s", ident.DID, repo)

		var tangledRepo *tangled.Repo
		tangledRepo, err = client.GetRepo(ctx, repoURI)
		if err != nil {
			return fmt.Errorf("get repo %s/%s: %w", handle, repo, err)
		}

		name := tangledRepo.Value.Name
		if name == "" {
			name = repo
		}

		result := repoItem{
			Name:        name,
			Author:      handle,
			URI:         repoURI,
			Knot:        tangledRepo.Value.Knot,
			Description: tangledRepo.Value.Description,
			CreatedAt:   tangledRepo.Value.CreatedAt,
			RepoDid:     tangledRepo.Value.RepoDid,
		}
		return output(result, func(item repoItem) {
			fmt.Printf("Name:        %s\n", item.Name)
			fmt.Printf("Description: %s\n", item.Description)
			fmt.Printf("URI:         %s\n", item.URI)
			fmt.Printf("Knot:        %s\n", item.Knot)
			fmt.Printf("Created:     %s\n", item.CreatedAt)
			if item.RepoDid != "" {
				fmt.Printf("Repo DID:    %s\n", item.RepoDid)
			}
		})

	},
}
