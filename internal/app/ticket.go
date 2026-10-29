package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func decodeTicket(raw json.RawMessage, pull bool) (tangled.Ticket, error) {
	var ticket tangled.Ticket
	if err := json.Unmarshal(raw, &ticket); err != nil {
		return tangled.Ticket{}, err
	}
	if ticket.Type != tangled.IssueCollection {
		return tangled.Ticket{}, fmt.Errorf("unexpected ticket type %q", ticket.Type)
	}
	if (ticket.TargetBranch != "") != pull {
		return tangled.Ticket{}, fmt.Errorf("ticket is not the requested resource kind")
	}
	return ticket, nil
}

func (s *Service) readTicket(ctx context.Context, repo *tangled.Repo, rkey string, pull bool) (*tangled.ListItem, tangled.Ticket, error) {
	if _, err := syntax.ParseRecordKey(rkey); err != nil {
		return nil, tangled.Ticket{}, err
	}
	uri := "at://" + stringValue(repo.Value.RepoDid) + "/" + tangled.IssueCollection + "/" + rkey
	item, err := s.appview.GetTicket(ctx, uri)
	if err != nil {
		return nil, tangled.Ticket{}, err
	}
	if item == nil || item.URI != uri {
		return nil, tangled.Ticket{}, fmt.Errorf("appview returned a different ticket URI")
	}
	result := *item
	if s.fosterKnot != "" {
		native, err := s.nativeSocial(ctx, repo)
		if err != nil {
			return nil, tangled.Ticket{}, err
		}
		if !native {
			found, err := s.knot.NewPublic(s.fosterKnot).GetRecord(ctx, stringValue(repo.Value.RepoDid), tangled.IssueCollection, rkey)
			if err != nil {
				return nil, tangled.Ticket{}, err
			}
			if found.URI != uri {
				return nil, tangled.Ticket{}, fmt.Errorf("foster returned a different ticket URI")
			}
			result.Value, err = json.Marshal(found.Value)
			if err != nil {
				return nil, tangled.Ticket{}, err
			}
			// The foster version replaces both the body and its version identifier.
			result.CID = ""
			if found.CID != nil {
				result.CID = found.CID.String()
			}
		}
	}
	ticket, err := decodeTicket(result.Value, pull)
	if err != nil {
		return nil, tangled.Ticket{}, err
	}
	result.State = ticket.State
	return &result, ticket, nil
}
