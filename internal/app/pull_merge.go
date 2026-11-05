package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (s *Service) MergePull(ctx context.Context, t Target, rkey string) (*MergePullResult, error) {
	repo, err := s.resolveVerifiedRepo(ctx, t)
	if err != nil {
		return nil, err
	}
	if err := s.requireNative(ctx, repo); err != nil {
		return nil, err
	}
	did := stringValue(repo.Value.RepoDid)
	item, ticket, err := s.readTicket(ctx, repo, rkey, true)
	if err != nil {
		return nil, err
	}
	if _, err := syntax.ParseCID(item.CID); err != nil {
		return nil, fmt.Errorf("ticket has no valid CID: %w", err)
	}
	if item.State != "open" {
		return nil, fmt.Errorf("only an open pull request can be merged")
	}
	found, err := s.knot.NewPublic(repo.Value.Knot).GetRecord(ctx, did, tangled.IssueCollection, rkey)
	if err != nil {
		return nil, err
	}
	if found.URI != item.URI || found.CID == nil || found.CID.String() != item.CID {
		return nil, fmt.Errorf("ticket changed or its authoritative version could not be confirmed; read it before merging")
	}
	raw, err := json.Marshal(found.Value)
	if err != nil {
		return nil, err
	}
	authoritative, err := decodeTicket(raw, true)
	if err != nil {
		return nil, err
	}
	if authoritative.State != "open" {
		return nil, fmt.Errorf("only an open pull request can be merged")
	}
	patches, err := s.appview.ListPatches(ctx, item.URI)
	if err != nil {
		return nil, err
	}
	if len(patches) == 0 {
		return nil, fmt.Errorf("pull request has no patch")
	}
	patch := patches[len(patches)-1]
	head := patch.Value.Head.OID
	source := did
	if patch.Value.Source != nil && patch.Value.Source.Repo != "" {
		source = patch.Value.Source.Repo
	}
	targetHead, err := s.appview.GetRef(ctx, did, "refs/heads/"+ticket.TargetBranch)
	if err != nil {
		return nil, err
	}
	pds, _, err := s.authenticatedPDS(ctx)
	if err != nil {
		return nil, err
	}
	token, err := pds.GetServiceAuth(ctx, serviceDID(repo.Value.Knot), "org.tangled.git.receivePack")
	if err != nil {
		return nil, err
	}
	const method = "sh.tangled.git.mergeCommit"
	input := map[string]any{"target": map[string]string{"$type": method + "#target", "repo": did, "branch": ticket.TargetBranch, "commit": targetHead}, "source": map[string]string{"$type": method + "#source", "repo": source, "commit": head}, "style": "rebase"}
	if err := s.appview.Merge(ctx, token, input); err != nil {
		return nil, fmt.Errorf("merge request failed; inspect the target branch before retrying: %w", explainMirrorError(err))
	}
	result := &MergePullResult{Rkey: rkey, Merged: true}
	latest, err := s.appview.ListPatches(ctx, item.URI)
	if err != nil || len(latest) == 0 || latest[len(latest)-1].CID != patch.CID || latest[len(latest)-1].Value.Head.OID != head {
		result.Warnings = []string{"merged code, but the latest PR round changed or could not be confirmed; ticket completion was skipped"}
		return result, nil
	}
	updated, err := s.editTicket(ctx, ticketEditInput{Target: t, Rkey: rkey, State: "completed", Pull: true, ExpectedCID: item.CID})
	if err != nil {
		result.Warnings = []string{fmt.Sprintf("merged code, but could not complete ticket: %v", err)}
		return result, nil
	}
	result.StatusRecorded = true
	result.Warnings = updated.Warnings
	return result, nil
}
