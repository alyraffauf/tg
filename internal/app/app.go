// Package app contains tg's frontend-independent operations and data types.
package app

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alyraffauf/tg/atproto"
	"github.com/alyraffauf/tg/internal/gitutil"
	"github.com/alyraffauf/tg/knot"
	"github.com/alyraffauf/tg/tangled"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/identity"
)

type Service struct {
	resolver              identityResolver
	appview               appviewClient
	sessions              sessionProvider
	auth                  *atproto.AuthManager
	git                   gitClient
	knot                  knotClientFactory
	spindle               spindleClientFactory
	knotOwnershipVerifier knotOwnershipVerifier
	httpClient            *http.Client
}

// DefaultKnot is used when repository creation does not specify a knot.
const DefaultKnot = knot.DefaultKnot

const defaultHTTPTimeout = 30 * time.Second

// New returns a Service with production defaults: the default atproto
// identity directory and the given appview host. The OAuth loopback redirect
// URI is configured per login (see SetOAuthCallbackURL) using an ephemeral
// port allocated when the callback server is bound.
func New(appviewHost string) *Service {
	return NewWithStreams(appviewHost, os.Stdout, os.Stderr)
}

// NewWithStreams creates production dependencies with configurable command
// output streams.
func NewWithStreams(appviewHost string, stdout, stderr io.Writer) *Service {
	httpClient := &http.Client{Timeout: defaultHTTPTimeout}
	resolver := &atproto.Resolver{Directory: identity.DefaultDirectory()}
	auth := atproto.NewAuthManagerWithClient(httpClient)
	return &Service{
		resolver: resolver,
		appview: &tangled.Tangled{
			Client: &atclient.APIClient{Client: httpClient, Host: appviewHost},
			Logger: slog.Default(),
		},
		sessions:              productionSessions{auth: auth, resolver: resolver, httpClient: httpClient},
		auth:                  auth,
		git:                   gitutil.NewClient(stdout, stderr),
		knot:                  productionKnotFactory{httpClient: httpClient},
		spindle:               productionSpindleFactory{httpClient: httpClient},
		knotOwnershipVerifier: knot.NewOwnershipVerifier(httpClient),
		httpClient:            httpClient,
	}
}

// SetOAuthCallbackURL configures the loopback redirect URI used for the next
// OAuth login. Should be called with the address of a freshly bound local
// listener before StartLogin.
func (s *Service) SetOAuthCallbackURL(callbackURL string) {
	s.auth.SetCallbackURL(callbackURL)
}

// SetAccount selects the account used by subsequent service operations.
func (s *Service) SetAccount(selector string) {
	s.auth.SetAccount(selector)
}
