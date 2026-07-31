package spindle

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func cborString(s string) []byte {
	l := len(s)
	switch {
	case l < 24:
		return append([]byte{0x60 | byte(l)}, s...)
	case l < 256:
		return append([]byte{0x78, byte(l)}, s...)
	default:
		return append([]byte{0x79, byte(l), byte(l >> 8)}, s...)
	}
}

func cborInt(n int) []byte {
	if n >= 0 {
		if n < 24 {
			return []byte{byte(n)}
		}
		return []byte{0x18, byte(n)}
	}
	v := -n - 1
	if v < 24 {
		return []byte{0x20 | byte(v)}
	}
	return []byte{0x38, byte(v)}
}

func cborMap(pairs ...[2][]byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte(0xa0 | byte(len(pairs)))
	for _, p := range pairs {
		buf.Write(p[0])
		buf.Write(p[1])
	}
	return buf.Bytes()
}

func TestDecodeLogEventControl(t *testing.T) {
	header := cborMap(
		[2][]byte{cborString("t"), cborString("#control")},
		[2][]byte{cborString("op"), cborInt(1)},
	)
	body := cborMap(
		[2][]byte{cborString("kind"), cborString("system")},
		[2][]byte{cborString("step"), cborInt(-1)},
		[2][]byte{cborString("time"), cborString("2026-07-31T06:34:08+03:00")},
		[2][]byte{cborString("status"), cborString("start")},
		[2][]byte{cborString("content"), cborString("Pull image")},
		[2][]byte{cborString("workflow"), cborString("build.yml")},
	)
	event, err := decodeLogEvent(append(header, body...))
	if err != nil {
		t.Fatalf("decodeLogEvent() error = %v", err)
	}
	if event.Type != "control" || event.Control == nil {
		t.Fatalf("decodeLogEvent() = %+v, want control event", event)
	}
	if event.Control.Workflow != "build.yml" || event.Control.Step != -1 {
		t.Fatalf("control = %+v", event.Control)
	}
	if event.Data != nil {
		t.Fatalf("unexpected data event")
	}
}

func TestDecodeLogEventData(t *testing.T) {
	header := cborMap(
		[2][]byte{cborString("t"), cborString("#data")},
		[2][]byte{cborString("op"), cborInt(1)},
	)
	body := cborMap(
		[2][]byte{cborString("step"), cborInt(0)},
		[2][]byte{cborString("time"), cborString("2026-07-31T06:34:10+03:00")},
		[2][]byte{cborString("stream"), cborString("stderr")},
		[2][]byte{cborString("content"), cborString("hint: Using 'master'")},
		[2][]byte{cborString("workflow"), cborString("build.yml")},
	)
	event, err := decodeLogEvent(append(header, body...))
	if err != nil {
		t.Fatalf("decodeLogEvent() error = %v", err)
	}
	if event.Type != "data" || event.Data == nil {
		t.Fatalf("decodeLogEvent() = %+v, want data event", event)
	}
	if event.Data.Stream != "stderr" || event.Data.Workflow != "build.yml" {
		t.Fatalf("data = %+v", event.Data)
	}
	if event.Control != nil {
		t.Fatalf("unexpected control event")
	}
}

func TestDecodeLogEventUnknownType(t *testing.T) {
	header := cborMap(
		[2][]byte{cborString("t"), cborString("#unknown")},
		[2][]byte{cborString("op"), cborInt(1)},
	)
	event, err := decodeLogEvent(append(header, cborMap()...))
	if err != nil {
		t.Fatalf("decodeLogEvent() error = %v", err)
	}
	if event != nil {
		t.Fatalf("decodeLogEvent() = %+v, want nil for unknown type", event)
	}
}

func TestSubscribePipelineLogsStreamsEventsAndCloses(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/xrpc/"+nsidSubscribeLogs.String() {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if request.URL.Query().Get("pipeline") != "3mrvk5dbnep22" {
			t.Fatalf("pipeline query = %q", request.URL.Query().Get("pipeline"))
		}
		if workflows := request.URL.Query()["workflows"]; len(workflows) != 2 || workflows[0] != "build.yml" || workflows[1] != "test.yml" {
			t.Fatalf("workflows query = %v", request.URL.Query()["workflows"])
		}
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		defer conn.Close()
		header := cborMap(
			[2][]byte{cborString("t"), cborString("#data")},
			[2][]byte{cborString("op"), cborInt(1)},
		)
		body := cborMap(
			[2][]byte{cborString("step"), cborInt(0)},
			[2][]byte{cborString("time"), cborString("2026-07-31T06:34:10+03:00")},
			[2][]byte{cborString("stream"), cborString("stdout")},
			[2][]byte{cborString("content"), cborString("building")},
			[2][]byte{cborString("workflow"), cborString("build.yml")},
		)
		if err := conn.WriteMessage(websocket.BinaryMessage, append(header, body...)); err != nil {
			t.Fatalf("write data frame: %v", err)
		}
		if err := conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)); err != nil {
			t.Fatalf("write close frame: %v", err)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	var events []PipelineLogEvent
	if err := client.SubscribePipelineLogs(context.Background(), "3mrvk5dbnep22", []string{"build.yml", "test.yml"}, func(event PipelineLogEvent) error {
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatalf("SubscribePipelineLogs() error = %v", err)
	}
	if len(events) != 1 || events[0].Data == nil || events[0].Data.Content != "building" {
		t.Fatalf("events = %+v", events)
	}
}

func TestSubscribePipelineLogsHonorsCancellation(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	upgraded := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		defer conn.Close()
		close(upgraded)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- client.SubscribePipelineLogs(ctx, "3mrvk5dbnep22", nil, func(PipelineLogEvent) error { return nil })
	}()
	<-upgraded
	cancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("SubscribePipelineLogs() error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SubscribePipelineLogs() did not return after cancellation")
	}
}
