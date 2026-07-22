package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
)

type identityResolver interface {
	ResolveHandle(context.Context, string) (*identity.Identity, error)
	ResolveDID(context.Context, string) (*identity.Identity, error)
	ResolvePDS(context.Context, string) (string, error)
}

type appviewClient interface {
	GetRepo(context.Context, string) (*tangled.Repo, error)
	ListRepos(context.Context, string) (*tangled.RepoList, error)
	ListIssues(context.Context, string, tangled.ListOpts) (*tangled.List, error)
	ListPulls(context.Context, string, tangled.ListOpts) (*tangled.List, error)
}

type pdsClient interface {
	PutRecord(context.Context, atproto.PutRecordInput) (string, string, error)
	DeleteRecord(context.Context, atproto.DeleteRecordInput) error
	UploadBlob(context.Context, []byte, string) (*atproto.Blob, error)
	GetRecord(context.Context, string, string, string) (*atproto.GetRecordOutput, error)
	ListAllRecords(context.Context, string, string, atproto.ListRecordsOpts) ([]atproto.RecordItem, error)
	GetServiceAuth(context.Context, string, string) (string, error)
}

type sessionProvider interface {
	AuthenticatedPDS(context.Context) (pdsClient, string, error)
	PublicPDS(context.Context, string) (pdsClient, string, error)
	APIClient(context.Context) (*atclient.APIClient, error)
}

type gitClient interface {
	CloneRepo(context.Context, gitutil.CloneRepoParams) error
	PushNewRepo(context.Context, gitutil.PushNewRepoParams) error
	CheckoutPatch(context.Context, gitutil.CheckoutPatchParams) error
	GeneratePatch(context.Context, string, string, string) ([]byte, error)
	CurrentBranch(context.Context, string) (string, error)
	DefaultBranch(context.Context, string) (string, error)
	DetectRepoFromCWD(context.Context) (*gitutil.RepoContext, error)
}

type knotClient interface {
	CreateRepo(context.Context, knot.CreateRepoInput) (string, error)
	DeleteRepo(context.Context, knot.DeleteRepoInput) error
	SetDefaultBranch(context.Context, knot.SetDefaultBranchInput) error
	Merge(context.Context, knot.MergeInput) error
}

type knotClientFactory interface {
	New(string, string) knotClient
}

type productionSessions struct {
	auth       *atproto.AuthManager
	resolver   identityResolver
	httpClient *http.Client
}

func (s productionSessions) AuthenticatedPDS(ctx context.Context) (pdsClient, string, error) {
	client, did, err := s.auth.APIClient(ctx)
	if err != nil {
		if isNotAuthenticated(err) {
			return nil, "", ErrNotAuthenticated
		}
		return nil, "", fmt.Errorf("resume auth session: %w", err)
	}
	return &atproto.ATProto{Client: client}, did.String(), nil
}

func (s productionSessions) PublicPDS(ctx context.Context, handle string) (pdsClient, string, error) {
	ident, err := s.resolver.ResolveHandle(ctx, handle)
	if err != nil {
		return nil, "", fmt.Errorf("resolve handle %q: %w", handle, err)
	}
	pdsURL, err := s.resolver.ResolvePDS(ctx, ident.DID.String())
	if err != nil {
		return nil, "", fmt.Errorf("resolve PDS for %q: %w", handle, err)
	}
	return &atproto.ATProto{Client: &atclient.APIClient{Client: s.httpClient, Host: pdsURL}}, ident.DID.String(), nil
}

func (s productionSessions) APIClient(ctx context.Context) (*atclient.APIClient, error) {
	client, _, err := s.auth.APIClient(ctx)
	if err != nil {
		if isNotAuthenticated(err) {
			return nil, ErrNotAuthenticated
		}
		return nil, fmt.Errorf("resume auth session: %w", err)
	}
	return client, nil
}

type productionKnotFactory struct {
	httpClient *http.Client
}

func (f productionKnotFactory) New(host, token string) knotClient {
	return knot.NewWithClient(host, token, f.httpClient)
}

func isNotAuthenticated(err error) bool {
	return errors.Is(err, ErrNotAuthenticated) || errors.Is(err, atproto.ErrNotAuthenticated)
}

var _ identityResolver = (*atproto.Resolver)(nil)
var _ appviewClient = (*tangled.Tangled)(nil)
var _ gitClient = (*gitutil.Client)(nil)
var _ pdsClient = (*atproto.ATProto)(nil)
var _ knotClient = (*knot.Client)(nil)
