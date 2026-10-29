package app

import "context"

func (s *Service) ViewIssue(ctx context.Context, t Target, rkey string) (*ViewResult, error) {
	return s.viewTicket(ctx, t, rkey, false)
}
func (s *Service) viewTicket(ctx context.Context, t Target, rkey string, pull bool) (*ViewResult, error) {
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	item, ticket, err := s.readTicket(ctx, repo, rkey, pull)
	if err != nil {
		return nil, err
	}
	state := ticket.State
	sourceBranch := ""
	if pull {
		if state == "completed" {
			state = "merged"
		}
		patches, err := s.appview.ListPatches(ctx, item.URI)
		if err != nil {
			return nil, err
		}
		if len(patches) > 0 && patches[len(patches)-1].Value.Source != nil {
			sourceBranch = patches[len(patches)-1].Value.Source.Branch
		}
	}
	body := ""
	if ticket.Body != nil {
		body = ticket.Body.Text
	}
	return &ViewResult{URI: item.URI, CID: item.CID, Rkey: rkey, Title: ticket.Title.Text, Body: body, State: state, Author: s.resolveAuthor(ctx, item.Author), CreatedAt: ticket.CreatedAt, TargetBranch: ticket.TargetBranch, SourceBranch: sourceBranch}, nil
}
