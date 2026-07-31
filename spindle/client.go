// Package spindle provides a client for Tangled CI spindle queries.
package spindle

import (
	"context"
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
type Workflow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
}

// Pipeline is a CI pipeline returned by a spindle.
type Pipeline struct {
	ID         string         `json:"id"`
	Commit     string         `json:"commit"`
	CreatedAt  string         `json:"createdAt,omitempty"`
	Repo       string         `json:"repo,omitempty"`
	SourceRepo string         `json:"sourceRepo,omitempty"`
	Trigger    map[string]any `json:"trigger"`
	Workflows  []Workflow     `json:"workflows"`
}

// QueryPipelinesOutput is the spindle response for sh.tangled.ci.queryPipelines.
type QueryPipelinesOutput struct {
	Pipelines []Pipeline `json:"pipelines"`
	Cursor    string     `json:"cursor,omitempty"`
	Total     int        `json:"total"`
}

// CancelPipelineInput is the argument to sh.tangled.ci.cancelPipeline.
type CancelPipelineInput struct {
	Pipeline  string   `json:"pipeline"`
	Repo      string   `json:"repo"`
	Workflows []string `json:"workflows,omitempty"`
}

// TriggerPipelineInput is the argument to sh.tangled.ci.triggerPipeline.
type TriggerPipelineInput struct {
	Repo      string        `json:"repo"`
	Trigger   ManualTrigger `json:"trigger"`
	Workflows []string      `json:"workflows,omitempty"`
}

// ManualTrigger describes a manually requested pipeline trigger.
type ManualTrigger struct {
	LexiconTypeID string `json:"$type"`
	SHA           string `json:"sha"`
	Ref           string `json:"ref,omitempty"`
}

// TriggerPipelineOutput is returned after creating a manual pipeline.
type TriggerPipelineOutput struct {
	Pipeline string `json:"pipeline"`
}

// TriggerPipeline starts a manual pipeline for a commit.
func (c *Client) TriggerPipeline(ctx context.Context, input TriggerPipelineInput) (*TriggerPipelineOutput, error) {
	var output TriggerPipelineOutput
	if err := c.Post(ctx, syntax.NSID("sh.tangled.ci.triggerPipeline"), input, &output); err != nil {
		return nil, fmt.Errorf("trigger pipeline: %w", err)
	}
	return &output, nil
}

// GetPipeline fetches one pipeline by its spindle-local ID.
func (c *Client) GetPipeline(ctx context.Context, pipelineID string) (*Pipeline, error) {
	var pipeline Pipeline
	if err := c.Get(ctx, syntax.NSID("sh.tangled.ci.getPipeline"), map[string]any{"pipeline": pipelineID}, &pipeline); err != nil {
		return nil, fmt.Errorf("get pipeline %q: %w", pipelineID, err)
	}
	return &pipeline, nil
}

// CancelPipeline cancels every workflow in a pipeline, or only Workflows when supplied.
func (c *Client) CancelPipeline(ctx context.Context, input CancelPipelineInput) error {
	if err := c.Post(ctx, syntax.NSID("sh.tangled.ci.cancelPipeline"), input, nil); err != nil {
		return fmt.Errorf("cancel pipeline %q: %w", input.Pipeline, err)
	}
	return nil
}

// QueryPipelines fetches one page of pipelines for repoDID.
func (c *Client) QueryPipelines(ctx context.Context, repoDID, cursor string) (*QueryPipelinesOutput, error) {
	params := map[string]any{"repo": repoDID, "limit": 250}
	if cursor != "" {
		params["cursor"] = cursor
	}

	var output QueryPipelinesOutput
	if err := c.Get(ctx, syntax.NSID("sh.tangled.ci.queryPipelines"), params, &output); err != nil {
		return nil, fmt.Errorf("query pipelines for %q: %w", repoDID, err)
	}
	return &output, nil
}

// QueryLatestPipeline fetches the most recent pipeline for repoDID.
func (c *Client) QueryLatestPipeline(ctx context.Context, repoDID string) (*QueryPipelinesOutput, error) {
	params := map[string]any{"repo": repoDID, "limit": 1}

	var output QueryPipelinesOutput
	if err := c.Get(ctx, syntax.NSID("sh.tangled.ci.queryPipelines"), params, &output); err != nil {
		return nil, fmt.Errorf("query latest pipeline for %q: %w", repoDID, err)
	}
	return &output, nil
}
