package spindle

import (
	"bytes"
	"testing"
)

func cborString(s string) []byte {
	var buf bytes.Buffer
	l := len(s)
	if l < 24 {
		buf.WriteByte(0x60 | byte(l))
	} else {
		buf.WriteByte(0x78)
		buf.WriteByte(byte(l))
	}
	buf.WriteString(s)
	return buf.Bytes()
}

func cborInt(n int) []byte {
	if n >= 0 {
		return []byte{byte(n)}
	}
	return []byte{0x20 | byte(-n-1)}
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
