// Package spindle provides a client for Tangled CI spindle queries.
package spindle

import (
	"context"
	"fmt"
	"net/http"
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
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, fmt.Errorf("spindle host is empty")
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	return &Client{APIClient: &atclient.APIClient{Client: httpClient, Host: host}}, nil
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
