package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alyraffauf/tg/internal/tangledlex"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (s *Service) requireNative(ctx context.Context, repo *tangled.Repo) error {
	native, err := s.nativeSocial(ctx, repo)
	if err != nil {
		return err
	}
	if !native {
		return fmt.Errorf("this operation requires a knot with social writes")
	}

	return nil
}
func (s *Service) socialProcedure(ctx context.Context, repo *tangled.Repo, method string, input map[string]any) (*knot.Receipt, []string, error) {
	// Provisioning can edit policy before publishing a declaration. Resolved
	// existing repositories always have a declaration URI and must verify it.
	if repo.DeclarationURI != "" {
		if err := s.verifyRepositoryClaim(ctx, repo); err != nil {
			return nil, nil, err
		}
	}
	if record, ok := input["record"]; ok {
		collection, _ := input["collection"].(string)
		if err := tangledlex.ValidateRecord(collection, record); err != nil {
			return nil, nil, err
		}
	}
	if err := s.requireNative(ctx, repo); err != nil {
		return nil, nil, err
	}
	pds, actor, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, nil, err
	}
	token, err := pds.GetServiceAuth(ctx, stringValue(repo.Value.RepoDid), method)
	if err != nil {
		return nil, nil, fmt.Errorf("authorize %s; renew OAuth grants with tg auth login if needed: %w", method, err)
	}
	input["repo"] = stringValue(repo.Value.RepoDid)
	var receipt knot.Receipt
	if err := s.knot.New(repo.Value.Knot, token).Procedure(ctx, method, input, &receipt); err != nil {
		return nil, nil, err
	}
	if receipt.URI == "" || receipt.CID == "" {
		return nil, nil, fmt.Errorf("knot accepted %s but omitted its record receipt; inspect the repository before retrying", method)
	}
	uri, err := syntax.ParseATURI(receipt.URI)
	collection, _ := input["collection"].(string)
	if method == "org.tangled.review.submitPatch" {
		collection = "org.tangled.review.patch"
	}
	if err != nil || uri.Authority().String() != stringValue(repo.Value.RepoDid) || uri.Collection().String() != collection {
		return nil, nil, fmt.Errorf("knot accepted %s but returned an inconsistent receipt; inspect before retrying", method)
	}
	if key, ok := input["rkey"].(string); ok && uri.RecordKey().String() != key {
		return nil, nil, fmt.Errorf("knot accepted %s but returned a different record key; inspect before retrying", method)
	}
	if _, err := syntax.ParseCID(receipt.CID); err != nil {
		return nil, nil, fmt.Errorf("knot accepted %s but returned an invalid CID; inspect before retrying", method)
	}
	warnings := s.finishWrite(ctx, pds, actor, receipt)
	return &receipt, warnings, nil
}
func (s *Service) finishWrite(ctx context.Context, pds pdsClient, actor string, receipt knot.Receipt) []string {
	// The mutation has succeeded. Bound follow-up work so callers receive its
	// receipt even when the PDS or indexer is unavailable.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var warnings []string
	if err := s.queueAttestation(ctx, pds, actor, receipt); err != nil {
		warnings = append(warnings, fmt.Sprintf("saved %s (%s), but attestation is pending: %v", receipt.URI, receipt.CID, err))
	}
	if err := s.appview.AwaitRecord(ctx, receipt.URI, receipt.CID); err != nil {
		warnings = append(warnings, fmt.Sprintf("saved %s, but Bobbin has not confirmed indexing: %v", receipt.URI, err))
	}
	return warnings
}

type ticketEditInput struct {
	ExpectedCID string
	Target      Target
	Rkey        string
	Title       *string
	Body        *string
	State       string
	Pull        bool
}

func (s *Service) editTicket(ctx context.Context, input ticketEditInput) (*UpdatedRecordResult, error) {
	t, rkey, title, body, state := input.Target, input.Rkey, input.Title, input.Body, input.State
	repo, err := s.resolveRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	native, err := s.nativeSocial(ctx, repo)
	if err != nil {
		return nil, err
	}
	if !native {
		return nil, fmt.Errorf("this operation requires a knot with social writes; legacy PDS ingestion is not verified")
	}
	found, err := s.knot.NewPublic(repo.Value.Knot).GetRecord(ctx, stringValue(repo.Value.RepoDid), tangled.IssueCollection, rkey)
	if err != nil {
		return nil, err
	}
	if found.CID == nil {
		return nil, fmt.Errorf("knot response omitted record CID")
	}
	if input.ExpectedCID != "" && found.CID.String() != input.ExpectedCID {
		return nil, fmt.Errorf("ticket changed since the expected CID; read it before retrying")
	}
	data, err := json.Marshal(found.Value)
	if err != nil {
		return nil, err
	}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}
	if record["$type"] != tangled.IssueCollection {
		return nil, fmt.Errorf("record is not a ticket")
	}
	targetBranch, _ := record["targetBranch"].(string)
	if (targetBranch != "") != input.Pull {
		return nil, fmt.Errorf("ticket is not the requested resource kind")
	}
	if legacyTitle, ok := record["title"].(string); ok {
		record["title"] = tangled.Markdown{Type: "org.tangled.markup.markdown#inline", Text: legacyTitle}
	}

	if title != nil {
		if strings.TrimSpace(*title) == "" {
			return nil, fmt.Errorf("title must not be empty")
		}
		record["title"] = tangled.Markdown{Type: "org.tangled.markup.markdown#inline", Text: *title}
	}
	if body != nil {
		if strings.TrimSpace(*body) == "" {
			delete(record, "body")
		} else {
			old, _ := record["body"].(map[string]any)
			if old == nil {
				old = map[string]any{}
			}
			old["$type"] = "org.tangled.markup.markdown"
			old["text"] = *body
			record["body"] = old
		}
	}
	if state != "" {
		record["state"] = state
	}
	receipt, warnings, err := s.socialProcedure(ctx, repo, "com.atproto.repo.putRecord", map[string]any{"collection": tangled.IssueCollection, "rkey": rkey, "swapRecord": found.CID.String(), "record": record})
	if err != nil {
		return nil, err
	}
	return &UpdatedRecordResult{Rkey: rkey, URI: receipt.URI, CID: receipt.CID, Warnings: warnings}, nil
}
func ticketRecord(title, body, target string) tangled.Ticket {
	record := tangled.Ticket{Type: tangled.IssueCollection, Title: tangled.Markdown{Type: "org.tangled.markup.markdown#inline", Text: title}, State: "open", TargetBranch: target, CreatedAt: nowString()}
	if strings.TrimSpace(body) != "" {
		record.Body = &tangled.Markdown{Type: "org.tangled.markup.markdown", Text: body}
	}
	return record
}
func (s *Service) eraseSocial(ctx context.Context, repo *tangled.Repo, receipt knot.Receipt) error {
	pds, _, err := s.authenticatedPDS(ctx)
	if err != nil {
		return err
	}
	method := "com.atproto.repo.deleteRecord"
	token, err := pds.GetServiceAuth(ctx, stringValue(repo.Value.RepoDid), method)
	if err != nil {
		return err
	}
	return s.knot.New(repo.Value.Knot, token).Procedure(ctx, method, map[string]any{"repo": stringValue(repo.Value.RepoDid), "collection": tangled.IssueCollection, "rkey": extractRKey(receipt.URI), "swapRecord": receipt.CID}, nil)
}

func isDefiniteRejection(err error) bool {
	var e *atclient.APIError
	return errors.As(err, &e) && e.StatusCode >= 400 && e.StatusCode < 500 && e.StatusCode != 408 && e.StatusCode != 429
}

func firstCID(values []string) string {
	if len(values) > 0 {
		return values[0]
	}
	return ""
}
