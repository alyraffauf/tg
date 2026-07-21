package tangled

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestListOptsParams(t *testing.T) {
	tests := []struct {
		name    string
		opts    ListOpts
		subject string
		cursor  string
		want    map[string]any
	}{
		{
			name:    "defaults",
			subject: "did:plc:repo",
			want:    map[string]any{"subject": "did:plc:repo", "limit": int64(50)},
		},
		{
			name:    "all options with cursor",
			opts:    ListOpts{Author: "did:plc:alice", State: "open", Limit: 10, Order: "asc"},
			subject: "did:plc:repo",
			cursor:  "cursor1",
			want: map[string]any{
				"subject": "did:plc:repo",
				"author":  "did:plc:alice",
				"state":   "open",
				"limit":   int64(10),
				"order":   "asc",
				"cursor":  "cursor1",
			},
		},
		{
			name:    "empty values omitted",
			opts:    ListOpts{State: "closed"},
			subject: "did:plc:repo",
			want: map[string]any{
				"subject": "did:plc:repo",
				"state":   "closed",
				"limit":   int64(50),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.opts.params(tt.subject, tt.cursor); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFetchAllPages(t *testing.T) {
	strptr := func(s string) *string { return &s }

	t.Run("single page", func(t *testing.T) {
		calls := 0
		items, err := fetchAllPages(context.Background(), func(_ context.Context, cursor string) ([]string, *string, error) {
			calls++
			if cursor != "" {
				t.Errorf("first fetch got cursor %q, want empty", cursor)
			}
			return []string{"a", "b"}, nil, nil
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !reflect.DeepEqual(items, []string{"a", "b"}) {
			t.Fatalf("got %v, want [a b]", items)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1", calls)
		}
	})

	t.Run("multiple pages", func(t *testing.T) {
		pages := map[string][]string{
			"":        {"a", "b"},
			"cursor1": {"c"},
			"cursor2": {"d"},
		}
		next := map[string]*string{"": strptr("cursor1"), "cursor1": strptr("cursor2"), "cursor2": nil}
		var gotCursors []string

		items, err := fetchAllPages(context.Background(), func(_ context.Context, cursor string) ([]string, *string, error) {
			gotCursors = append(gotCursors, cursor)
			pageItems, ok := pages[cursor]
			if !ok {
				t.Errorf("unexpected cursor %q", cursor)
				return nil, nil, nil
			}
			return pageItems, next[cursor], nil
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if !reflect.DeepEqual(items, []string{"a", "b", "c", "d"}) {
			t.Fatalf("got %v, want [a b c d]", items)
		}
		if !reflect.DeepEqual(gotCursors, []string{"", "cursor1", "cursor2"}) {
			t.Fatalf("cursors = %v, want [ cursor1 cursor2]", gotCursors)
		}
	})

	t.Run("empty first page", func(t *testing.T) {
		items, err := fetchAllPages(context.Background(), func(_ context.Context, _ string) ([]string, *string, error) {
			return nil, nil, nil
		})
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("got %v, want no items", items)
		}
	})

	t.Run("error propagates", func(t *testing.T) {
		errFetch := errors.New("fetch failed")
		items, err := fetchAllPages(context.Background(), func(_ context.Context, cursor string) ([]string, *string, error) {
			if cursor != "" {
				return nil, nil, errFetch
			}
			return []string{"a"}, strptr("cursor1"), nil
		})
		if !errors.Is(err, errFetch) {
			t.Fatalf("error = %v, want %v", err, errFetch)
		}
		if items != nil {
			t.Fatalf("got %v, want nil items on error", items)
		}
	})

	t.Run("runaway cursor", func(t *testing.T) {
		calls := 0
		_, err := fetchAllPages(context.Background(), func(_ context.Context, _ string) ([]string, *string, error) {
			calls++
			return []string{"a"}, strptr("next"), nil
		})
		if err == nil {
			t.Fatal("expected an error after exceeding the page limit")
		}
		if calls != maxPaginationPages {
			t.Fatalf("calls = %d, want %d", calls, maxPaginationPages)
		}
	})
}
