package tangled

import (
	"encoding/json"
	"testing"
)

func FuzzTicketDecoding(f *testing.F) {
	f.Add([]byte(`{"$type":"org.tangled.track.ticket","title":{"text":"Hello"}}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		var value Ticket
		_ = json.Unmarshal(data, &value)
	})
}
