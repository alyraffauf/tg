// Package spindle provides a client for Tangled CI spindle queries.
package spindle

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// Client queries a Tangled CI spindle.
type Client struct {
	*atclient.APIClient
}

// New creates a spindle client. A spindle record may contain either a host
// name or a complete HTTP URL.
func New(host string, httpClient *http.Client) (*Client, error) {
	return newClient(host, httpClient, nil)
}

// NewWithToken creates a spindle client authenticated with a service-auth JWT.
func NewWithToken(host, token string, httpClient *http.Client) (*Client, error) {
	return newClient(host, httpClient, bearerAuth(token))
}

func newClient(host string, httpClient *http.Client, auth atclient.AuthMethod) (*Client, error) {
	serviceURL, err := parseServiceURL(host)
	if err != nil {
		return nil, err
	}
	return &Client{APIClient: &atclient.APIClient{Client: httpClient, Host: serviceURL.String(), Auth: auth}}, nil
}

// ServiceDID returns the did:web identifier used as a spindle's service-auth audience.
func ServiceDID(host string) (string, error) {
	serviceURL, err := parseServiceURL(host)
	if err != nil {
		return "", err
	}
	return "did:web:" + strings.ReplaceAll(serviceURL.Host, ":", "%3A"), nil
}

func parseServiceURL(host string) (*url.URL, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, fmt.Errorf("spindle host is empty")
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	serviceURL, err := url.Parse(host)
	if err != nil || serviceURL.Host == "" {
		return nil, fmt.Errorf("invalid spindle host %q", host)
	}
	return serviceURL, nil
}

type bearerAuth string

func (b bearerAuth) DoWithAuth(client *http.Client, request *http.Request, _ syntax.NSID) (*http.Response, error) {
	request.Header.Set("Authorization", "Bearer "+string(b))
	return client.Do(request)
}

// Workflow is one workflow executed by a pipeline.
