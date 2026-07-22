package knot

import (
	"net/http"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// DefaultKnot is the public Tangled knot used when none is specified.
const DefaultKnot = "knot1.tangled.sh"

// Client calls Tangled knot procedures, authenticated with a PDS-minted
// service-auth JWT (Bearer).
type Client struct {
	*atclient.APIClient
}

// New returns a Client for host, authenticated with a service-auth token.
func New(host, token string) *Client {
	return NewWithClient(host, token, http.DefaultClient)
}

// NewWithClient returns a Client using httpClient for requests.
func NewWithClient(host, token string, httpClient *http.Client) *Client {
	return &Client{
		APIClient: &atclient.APIClient{
			Client: httpClient,
			Host:   "https://" + host,
			Auth:   bearerAuth(token),
		},
	}
}

// bearerAuth is a Bearer-token AuthMethod for service-auth JWTs.
type bearerAuth string

func (b bearerAuth) DoWithAuth(c *http.Client, req *http.Request, _ syntax.NSID) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+string(b))
	return c.Do(req)
}
