package app

import (
	"context"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
)

// ViewRepo fetches a single repository record.
func (s *Service) ViewRepo(ctx context.Context, t Target) (*RepoItem, error) {
	tangledRepo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	name := stringValue(tangledRepo.Value.Name)
	if name == "" {
		name = t.Repo
	}
	return &RepoItem{
		Name:        name,
		Author:      s.ownerHandle(ctx, tangledRepo.OwnerDID),
		URI:         tangledRepo.URI,
		Knot:        tangledRepo.Value.Knot,
		Description: stringValue(tangledRepo.Value.Description),
		CreatedAt:   tangledRepo.Value.CreatedAt,
		RepoDid:     stringValue(tangledRepo.Value.RepoDid),
	}, nil
}

var readmeNames = []string{
	"README.md",
	"readme.md",
	"README.markdown",
	"readme.markdown",
}

func (s *Service) RepoReadme(ctx context.Context, t Target) (string, error) {
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return "", err
	}
	repoDID := stringValue(repo.Value.RepoDid)
	if repoDID == "" {
		return "", fmt.Errorf("repository %q has no DID", t.String())
	}

	for _, name := range readmeNames {
		contents, err := s.appview.GetBlob(ctx, repoDID, name)
		if err != nil {
			if tangled.IsNotFound(err) {
				continue
			}
			return "", fmt.Errorf("read %q: %w", name, err)
		}
		return string(contents), nil
	}
	return "", nil
}
