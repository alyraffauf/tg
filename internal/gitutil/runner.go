package gitutil

import (
	"context"
	"io"
	"os/exec"
)

// Client runs git commands with configurable output sinks.
type Client struct {
	Stdout io.Writer
	Stderr io.Writer
}

// NewClient returns a git client connected to the supplied output sinks.
// Nil sinks discard command output.
func NewClient(stdout, stderr io.Writer) *Client {
	return &Client{Stdout: stdout, Stderr: stderr}
}

func (c *Client) writers() (io.Writer, io.Writer) {
	if c == nil {
		return io.Discard, io.Discard
	}
	stdout, stderr := c.Stdout, c.Stderr
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return stdout, stderr
}

// run executes a command in the foreground, connected to the client's sinks.
func (c *Client) run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr = c.writers()
	return cmd.Run()
}

// runIn is like run but sets the working directory to dir.
func (c *Client) runIn(dir string, ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout, cmd.Stderr = c.writers()
	return cmd.Run()
}

var defaultClient = NewClient(nil, nil)

func run(ctx context.Context, name string, args ...string) error {
	return defaultClient.run(ctx, name, args...)
}

func runIn(dir string, ctx context.Context, name string, args ...string) error {
	return defaultClient.runIn(dir, ctx, name, args...)
}
