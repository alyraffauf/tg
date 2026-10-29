package tangledlex

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestValidatePinnedRecords(t *testing.T) {
	tests := []struct {
		name, collection, record string
		valid                    bool
	}{
		{"declaration", "org.tangled.repo.declaration", `{"$type":"org.tangled.repo.declaration","repo":"did:plc:repository","createdAt":"2026-10-01T00:00:00Z"}`, true},
		{"missing repo", "org.tangled.repo.declaration", `{"$type":"org.tangled.repo.declaration","createdAt":"2026-10-01T00:00:00Z"}`, false},
		{"ticket", "org.tangled.track.ticket", `{"$type":"org.tangled.track.ticket","title":{"$type":"org.tangled.markup.markdown#inline","text":"Hello"},"state":"open","createdAt":"2026-10-01T00:00:00Z"}`, true},
		{"plain title", "org.tangled.track.ticket", `{"$type":"org.tangled.track.ticket","title":"Hello","state":"open","createdAt":"2026-10-01T00:00:00Z"}`, false},
		{"bad time", "org.tangled.repo.declaration", `{"$type":"org.tangled.repo.declaration","repo":"did:plc:repository","createdAt":"yesterday"}`, false},
		{"legacy", "sh.tangled.repo", `{"$type":"sh.tangled.repo"}`, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var record any
			if err := json.Unmarshal([]byte(test.record), &record); err != nil {
				t.Fatal(err)
			}
			err := ValidateRecord(test.collection, record)
			if (err == nil) != test.valid {
				t.Fatalf("validation=%v valid=%v", err, test.valid)
			}
		})
	}
}
func TestGeneratedRecordRoundTrip(t *testing.T) {
	original := RepoDeclaration{LexiconTypeID: "org.tangled.repo.declaration", Repo: "did:plc:repository", CreatedAt: "2026-10-01T00:00:00Z"}
	var data bytes.Buffer
	if err := original.MarshalCBOR(&data); err != nil {
		t.Fatal(err)
	}
	var decoded RepoDeclaration
	if err := decoded.UnmarshalCBOR(&data); err != nil {
		t.Fatal(err)
	}
	if decoded.Repo != original.Repo || decoded.CreatedAt != original.CreatedAt {
		t.Fatalf("record=%+v", decoded)
	}
}
