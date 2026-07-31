package spindle

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// SubscribePipelineLogs streams log events from a pipeline over WebSocket.
// The spindle closes the connection when logs are fully delivered.
func (c *Client) SubscribePipelineLogs(ctx context.Context, pipelineID string, workflows []string, onEvent func(PipelineLogEvent) error) error {
	u, err := url.Parse(c.Host)
	if err != nil {
		return fmt.Errorf("parse spindle host: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return fmt.Errorf("spindle host must be http(s), got %q", u.Scheme)
	}
	u.Path = "/xrpc/" + nsidSubscribeLogs.String()
	query := url.Values{"pipeline": []string{pipelineID}}
	if len(workflows) > 0 {
		query["workflows"] = workflows
	}
	u.RawQuery = query.Encode()

	dialer := websocket.Dialer{HandshakeTimeout: 30 * time.Second}
	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("connect pipeline log subscription: %w", err)
	}
	defer conn.Close()

	// Interrupt a read blocked on the connection when ctx is cancelled.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-done:
		}
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
				return nil
			}
			return fmt.Errorf("read pipeline log: %w", err)
		}
		event, err := decodeLogEvent(data)
		if err != nil {
			return err
		}
		if event == nil {
			continue
		}
		if err := onEvent(*event); err != nil {
			return err
		}
	}
}

// decodeLogEvent decodes one WebSocket frame into an event. Frames with fewer
// than two CBOR maps or an unknown header type are skipped and return nil.
func decodeLogEvent(data []byte) (*PipelineLogEvent, error) {
	maps, err := decodeCBORMaps(data)
	if err != nil {
		return nil, fmt.Errorf("decode pipeline log frame: %w", err)
	}
	if len(maps) < 2 {
		return nil, nil
	}
	header, body := maps[0], maps[1]
	switch header["t"] {
	case "#control":
		return &PipelineLogEvent{
			Type: "control",
			Control: &PipelineLogControl{
				Kind:     mapString(body, "kind"),
				Step:     mapInt(body, "step"),
				Time:     mapString(body, "time"),
				Status:   mapString(body, "status"),
				Content:  mapString(body, "content"),
				Workflow: mapString(body, "workflow"),
				Command:  mapStringOpt(body, "command"),
			},
		}, nil
	case "#data":
		return &PipelineLogEvent{
			Type: "data",
			Data: &PipelineLogData{
				Step:     mapInt(body, "step"),
				Time:     mapString(body, "time"),
				Stream:   mapString(body, "stream"),
				Content:  mapString(body, "content"),
				Workflow: mapString(body, "workflow"),
			},
		}, nil
	default:
		return nil, nil
	}
}
