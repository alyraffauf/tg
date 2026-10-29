package tangledlex

import (
	"encoding/json"
	"testing"
)

func FuzzValidateRecord(f *testing.F) {
	f.Add([]byte(`{"$type":"org.tangled.track.ticket"}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		var value any
		if json.Unmarshal(data, &value) == nil {
			_ = ValidateRecord("org.tangled.track.ticket", value)
		}
	})
}
