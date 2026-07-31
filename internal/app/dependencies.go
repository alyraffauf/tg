package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/spindle"
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
	Search(context.Context, string, int64) (*tangled.SearchResult, error)
	CountStars(context.Context, string) (int64, error)
}

type pdsClient interface {
	PutRecord(context.Context, atproto.PutRecordInput) (string, string, error)
	DeleteRecord(context.Context, atproto.DeleteRecordInput) error
	UploadBlob(context.Context, []byte, string) (*atproto.Blob, error)
	GetRecord(context.Context, string, string, string) (*atproto.GetRecordOutput, error)
	ListRecords(context.Context, string, string, atproto.ListRecordsOpts) (*atproto.ListRecordsOutput, error)
	ListAllRecords(context.Context, string, string, atproto.ListRecordsOpts) ([]atproto.RecordItem, error)
	GetServiceAuth(context.Context, string, string) (string, error)
}

type sessionProvider interface {
	AuthenticatedPDS(context.Context) (pdsClient, string, error)
	PublicPDS(context.Context, string) (pdsClient, string, error)
	APIClient(context.Context) (*atclient.APIClient, error)
	OAuthSessionHasScope(context.Context, string) (hasScope, isOAuth bool, err error)
}

type gitClient interface {
	CloneRepo(context.Context, gitutil.CloneRepoParams) error
	PushNewRepo(context.Context, gitutil.PushNewRepoParams) error
	CheckoutPatch(context.Context, gitutil.CheckoutPatchParams) error
	GeneratePatch(context.Context, string, string, string) ([]byte, error)
	CurrentBranch(context.Context, string) (string, error)
	ResolveCommit(context.Context, string, string) (string, error)
	DefaultBranch(context.Context, string) (string, error)
	DetectRepoCandidatesFromCWD(context.Context) ([]gitutil.RepoContext, error)
}

type knotClient interface {
	CreateRepo(context.Context, knot.CreateRepoInput) (string, error)
	DeleteRepo(context.Context, knot.DeleteRepoInput) error
	SetDefaultBranch(context.Context, knot.SetDefaultBranchInput) error
	GetDefaultBranch(context.Context, string) (*knot.DefaultBranch, error)
	Merge(context.Context, knot.MergeInput) error
}

type knotClientFactory interface {
	New(string, string) knotClient
	NewPublic(string) knotClient
}

type pipelineClient interface {
	QueryPipelines(context.Context, string, string) (*spindle.QueryPipelinesOutput, error)
	QueryLatestPipeline(context.Context, string) (*spindle.QueryPipelinesOutput, error)
	GetPipeline(context.Context, string) (*spindle.Pipeline, error)
	CancelPipeline(context.Context, spindle.CancelPipelineInput) error
	TriggerPipeline(context.Context, spindle.TriggerPipelineInput) (*spindle.TriggerPipelineOutput, error)
}

type spindleClientFactory interface {
	New(string) (pipelineClient, error)
	NewWithToken(string, string) (pipelineClient, error)
}

type knotOwnershipVerifier interface {
	Verify(ctx context.Context, host, expectedOwnerDID string) error
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

func (s productionSessions) OAuthSessionHasScope(ctx context.Context, scope string) (hasScope, isOAuth bool, err error) {
	return s.auth.OAuthSessionHasScope(ctx, scope)
}

type productionKnotFactory struct {
	httpClient *http.Client
}

type productionSpindleFactory struct {
	httpClient *http.Client
}

func (f productionSpindleFactory) New(host string) (pipelineClient, error) {
	return spindle.New(host, f.httpClient)
}

func (f productionSpindleFactory) NewWithToken(host, token string) (pipelineClient, error) {
	return spindle.NewWithToken(host, token, f.httpClient)
}

func (f productionKnotFactory) New(host, token string) knotClient {
	return knot.NewWithClient(host, token, f.httpClient)
}

func (f productionKnotFactory) NewPublic(host string) knotClient {
	return knot.NewPublicWithClient(host, f.httpClient)
}

func isNotAuthenticated(err error) bool {
	return errors.Is(err, ErrNotAuthenticated) || errors.Is(err, atproto.ErrNotAuthenticated)
}

var (
	_ identityResolver = (*atproto.Resolver)(nil)
	_ appviewClient    = (*tangled.Tangled)(nil)
	_ gitClient        = (*gitutil.Client)(nil)
	_ pdsClient        = (*atproto.ATProto)(nil)
	_ knotClient       = (*knot.Client)(nil)
	_ pipelineClient   = (*spindle.Client)(nil)
)
