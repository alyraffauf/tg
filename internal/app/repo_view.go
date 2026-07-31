package app

import "context"

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
		Author:      t.Handle,
		URI:         tangledRepo.URI,
		Knot:        tangledRepo.Value.Knot,
		Description: stringValue(tangledRepo.Value.Description),
		CreatedAt:   tangledRepo.Value.CreatedAt,
		RepoDid:     stringValue(tangledRepo.Value.RepoDid),
	}, nil
}
