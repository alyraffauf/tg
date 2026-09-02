package tangled

import (
	"encoding/json"
	"testing"

	"github.com/alyraffauf/tg/internal/tangledlex"
)

func FuzzIssueListItemDecoding(f *testing.F) {
	f.Add([]byte(`[{"uri":"at://did:plc:owner/sh.tangled.repo.issue/one","value":null}]`))
	f.Add([]byte(`[null]`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		var items []*tangledlex.RepoListIssues_IssueListItem
		if json.Unmarshal(data, &items) != nil {
			return
		}
		_, _ = issueListItems(items)
	})
}
