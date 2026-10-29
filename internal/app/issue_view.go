package app

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

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
	uri := "at://" + stringValue(repo.Value.RepoDid) + "/" + tangled.IssueCollection + "/" + rkey
	item, err := s.appview.GetTicket(ctx, uri)
	if err != nil {
		return nil, err
	}
	if item == nil || item.URI != uri {
		return nil, fmt.Errorf("appview returned a different ticket URI")
	}
	if s.fosterKnot != "" {
		capabilities, err := s.knot.NewPublic(repo.Value.Knot).DescribeKnot(ctx)
		if err != nil {
			return nil, err
		}
		if capabilities.Legacy || !slices.Contains(capabilities.Capabilities, "knot-social-writes") {
			found, err := s.knot.NewPublic(s.fosterKnot).GetRecord(ctx, stringValue(repo.Value.RepoDid), tangled.IssueCollection, rkey)
			if err != nil {
				return nil, err
			}
			if found.URI != uri {
				return nil, fmt.Errorf("foster returned a different ticket URI")
			}
			data, err := json.Marshal(found.Value)
			if err != nil {
				return nil, err
			}
			item.Value = data
			// The foster version replaces both the body and its version identifier.
			item.CID = ""
			if found.CID != nil {
				item.CID = found.CID.String()
			}
		}
	}
	var ticket tangled.Ticket
	if err := json.Unmarshal(item.Value, &ticket); err != nil {
		return nil, err
	}
	if (ticket.TargetBranch != "") != pull {
		return nil, fmt.Errorf("ticket is not the requested resource kind")
	}
	decoded, err := decodeIssue(item.Value)
	if err != nil {
		return nil, err
	}
	state := ticket.State
	sourceBranch := ""
	if pull {
		if state == "completed" {
			state = "merged"
		}
		patches, err := s.appview.ListPatches(ctx, uri)
		if err != nil {
			return nil, err
		}
		if len(patches) > 0 && patches[len(patches)-1].Value.Source != nil {
			sourceBranch = patches[len(patches)-1].Value.Source.Branch
		}
	}
	return &ViewResult{URI: item.URI, CID: item.CID, Rkey: rkey, Title: decoded.Title, Body: decoded.Body, State: state, Author: s.resolveAuthor(ctx, item.Author), CreatedAt: decoded.CreatedAt, TargetBranch: decoded.TargetBranch, SourceBranch: sourceBranch}, nil
}
