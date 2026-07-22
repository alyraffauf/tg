// Package app is tg's frontend-independent application layer. A Service
// bundles the atproto, appview, and auth dependencies and exposes every
// operation tg supports — resolving repositories, listing and mutating
// issues and pull requests, managing repos, strings, and SSH keys — as
// methods that take plain inputs and return typed domain structs.
//
// Frontends (the Cobra CLI in internal/cli, a future TUI or GUI) construct
// a Service, translate user intent into method calls, and render the
// returned structs however they see fit.
package app

import (
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
)

// Dependencies configures the external clients used by Service.
type Dependencies struct {
	Resolver   *atproto.Resolver
	Appview    *tangled.Tangled
	Auth       *atproto.AuthManager
	Git        *gitutil.Client
	HTTPClient *http.Client
}

// Service bundles the dependencies every tg operation needs.
type Service struct {
	// Resolver resolves handles and DIDs via the atproto identity directory.
	Resolver *atproto.Resolver
	// Appview is the read-only Tangled appview (bobbin) XRPC client.
	Appview *tangled.Tangled
	// Auth manages atproto sessions (OAuth and app-password) in the keyring.
	Auth *atproto.AuthManager
	// Git runs local git operations.
	Git *gitutil.Client
	// HTTPClient downloads pull request patches.
	HTTPClient *http.Client
}

// New returns a Service with production defaults: the default atproto
// identity directory, the given appview host, and an AuthManager using
// oauthCallbackURL for localhost OAuth redirects.
func New(appviewHost, oauthCallbackURL string) *Service {
	return NewWithStreams(appviewHost, oauthCallbackURL, os.Stdout, os.Stderr)
}

// NewWithStreams creates production dependencies with configurable command
// output streams.
func NewWithStreams(appviewHost, oauthCallbackURL string, stdout, stderr io.Writer) *Service {
	return NewWithDependencies(Dependencies{
		Resolver: &atproto.Resolver{Directory: identity.DefaultDirectory()},
		Appview: &tangled.Tangled{
			Client: &atclient.APIClient{Host: appviewHost},
			Logger: slog.Default(),
		},
		Auth:       atproto.NewAuthManager(oauthCallbackURL),
		Git:        gitutil.NewClient(stdout, stderr),
		HTTPClient: http.DefaultClient,
	})
}

// NewWithDependencies constructs a Service from explicit dependencies.
func NewWithDependencies(dependencies Dependencies) *Service {
	return &Service{
		Resolver:   dependencies.Resolver,
		Appview:    dependencies.Appview,
		Auth:       dependencies.Auth,
		Git:        dependencies.Git,
		HTTPClient: dependencies.HTTPClient,
	}
}

// SetAccount selects the account used by subsequent service operations.
func (s *Service) SetAccount(selector string) {
	s.Auth.SetAccount(selector)
}