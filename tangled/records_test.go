package tangled

import (
	"encoding/json"
	"testing"
)

func TestTicketTitleReadCompatibility(t *testing.T) {
	for _, title := range []string{`"legacy"`, `{"$type":"org.tangled.markup.markdown#inline","text":"legacy"}`} {
		var ticket Ticket
		if err := json.Unmarshal([]byte(`{"$type":"org.tangled.track.ticket","title":`+title+`}`), &ticket); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(ticket)
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := json.Unmarshal(encoded, &value); err != nil {
			t.Fatal(err)
		}
		if value["title"].(map[string]any)["text"] != "legacy" {
			t.Fatalf("ticket=%s", encoded)
		}
	}
	var ticket Ticket
	if err := json.Unmarshal([]byte(`{"title":17}`), &ticket); err == nil {
		t.Fatal("accepted a numeric title")
	}
}
