package app

import (
	"context"

	"github.com/alyraffauf/tg/tangled"
)

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
	labels, warnings := s.ticketLabels(ctx, stringValue(repo.Value.RepoDid), ticket.TangledLabels)
	return &ViewResult{URI: item.URI, CID: item.CID, Rkey: rkey, Title: ticket.Title.Text, Body: body, State: state, Author: s.resolveAuthor(ctx, item.Author), CreatedAt: ticket.CreatedAt, TargetBranch: ticket.TargetBranch, SourceBranch: sourceBranch, Labels: labels, Warnings: warnings}, nil
}

// ticketLabels names a ticket's applied labels. Definitions are only read when
// the ticket has labels.
func (s *Service) ticketLabels(ctx context.Context, repoDID string, tokens []tangled.LabelToken) ([]Label, []string) {
	if len(appliedLabelKeys(tokens)) == 0 {
		return []Label{}, nil
	}
	catalog, catalogWarnings := s.loadDisplayLabelCatalog(ctx, repoDID)
	labels, unresolvedKeys := catalog.appliedLabels(tokens)
	var warnings []string
	for _, warning := range catalogWarnings {
		warnings = append(warnings, formatRecordWarning(warning))
	}
	if len(unresolvedKeys) > 0 {
		warnings = append(warnings, unresolvedLabelMessage(unresolvedKeys))
	}
	return labels, warnings
}

func formatRecordWarning(warning RecordWarning) string {
	if warning.URI == "" {
		return warning.Error
	}
	return warning.URI + ": " + warning.Error
}
