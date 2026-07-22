// Package app contains tg's frontend-independent operations and data types.
package app

import (
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
)

type Service struct {
	resolver   identityResolver
	appview    appviewClient
	sessions   sessionProvider
	auth       *atproto.AuthManager
	git        gitClient
	knot       knotClientFactory
	httpClient *http.Client
}

// DefaultKnot is used when repository creation does not specify a knot.
const DefaultKnot = knot.DefaultKnot

// New returns a Service with production defaults: the default atproto
// identity directory, the given appview host, and an AuthManager using
// oauthCallbackURL for localhost OAuth redirects.
func New(appviewHost, oauthCallbackURL string) *Service {
	return NewWithStreams(appviewHost, oauthCallbackURL, os.Stdout, os.Stderr)
}

// NewWithStreams creates production dependencies with configurable command
// output streams.
func NewWithStreams(appviewHost, oauthCallbackURL string, stdout, stderr io.Writer) *Service {
	resolver := &atproto.Resolver{Directory: identity.DefaultDirectory()}
	auth := atproto.NewAuthManager(oauthCallbackURL)
	return &Service{
		resolver: resolver,
		appview: &tangled.Tangled{
			Client: &atclient.APIClient{Host: appviewHost},
			Logger: slog.Default(),
		},
		sessions:   productionSessions{auth: auth, resolver: resolver},
		auth:       auth,
		git:        gitutil.NewClient(stdout, stderr),
		knot:       productionKnotFactory{},
		httpClient: http.DefaultClient,
	}
}

// SetAccount selects the account used by subsequent service operations.
func (s *Service) SetAccount(selector string) {
	s.auth.SetAccount(selector)
}
