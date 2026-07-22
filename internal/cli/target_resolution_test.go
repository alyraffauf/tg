package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/alyraffauf/tg/internal/app"
)

func TestResolveHandleOrSelf(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		resolver  fakeAccountHandleResolver
		want      string
		wantCalls int
		wantError bool
	}{
		{
			name: "explicit handle",
			args: []string{"other.test"},
			resolver: fakeAccountHandleResolver{
				handle: "self.test",
			},
			want: "other.test",
		},
		{
			name: "authenticated user",
			resolver: fakeAccountHandleResolver{
				handle: "self.test",
			},
			want:      "self.test",
			wantCalls: 1,
		},
		{
			name: "authentication failure",
			resolver: fakeAccountHandleResolver{
				err: errors.New("not logged in"),
			},
			wantCalls: 1,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveHandleOrSelf(context.Background(), tt.args, &tt.resolver)
			if (err != nil) != tt.wantError {
				t.Fatalf("resolveHandleOrSelf() error = %v, want error %t", err, tt.wantError)
			}
			if got != tt.want {
				t.Fatalf("resolveHandleOrSelf() = %q, want %q", got, tt.want)
			}
			if tt.resolver.calls != tt.wantCalls {
				t.Fatalf("HandleOrSelf() calls = %d, want %d", tt.resolver.calls, tt.wantCalls)
			}
		})
	}
}

func TestResolveCloneTarget(t *testing.T) {
	tests := []struct {
		name      string
		arg       string
		resolver  fakeAccountHandleResolver
		want      app.Target
		wantCalls int
		wantError bool
	}{
		{
			name: "explicit handle and repo",
			arg:  "other.test/tg",
			want: app.Target{Handle: "other.test", Repo: "tg"},
		},
		{
			name: "authenticated user repo",
			arg:  "tg",
			resolver: fakeAccountHandleResolver{
				handle: "self.test",
			},
			want:      app.Target{Handle: "self.test", Repo: "tg"},
			wantCalls: 1,
		},
		{
			name: "authentication failure",
			arg:  "tg",
			resolver: fakeAccountHandleResolver{
				err: errors.New("not logged in"),
			},
			wantCalls: 1,
			wantError: true,
		},
		{
			name:      "empty repo",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveCloneTarget(context.Background(), tt.arg, &tt.resolver)
			if (err != nil) != tt.wantError {
				t.Fatalf("resolveCloneTarget() error = %v, want error %t", err, tt.wantError)
			}
			if got != tt.want {
				t.Fatalf("resolveCloneTarget() = %+v, want %+v", got, tt.want)
			}
			if tt.resolver.calls != tt.wantCalls {
				t.Fatalf("HandleOrSelf() calls = %d, want %d", tt.resolver.calls, tt.wantCalls)
			}
		})
	}
}

type fakeAccountHandleResolver struct {
	handle string
	err    error
	calls  int
}

func (r *fakeAccountHandleResolver) HandleOrSelf(context.Context, string) (string, error) {
	r.calls++
	return r.handle, r.err
}
