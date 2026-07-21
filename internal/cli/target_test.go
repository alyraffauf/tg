package cli

import "testing"

func TestParseHandleRepo(t *testing.T) {
	tests := []struct {
		name       string
		arg        string
		wantHandle string
		wantRepo   string
		wantErr    bool
	}{
		{name: "handle and repo", arg: "aly.codes/tg", wantHandle: "aly.codes", wantRepo: "tg"},
		{name: "did handle", arg: "did:plc:abc123/tg", wantHandle: "did:plc:abc123", wantRepo: "tg"},
		{name: "repo containing slash", arg: "aly.codes/a/b", wantHandle: "aly.codes", wantRepo: "a/b"},
		{name: "no slash", arg: "tg", wantErr: true},
		{name: "empty handle", arg: "/tg", wantErr: true},
		{name: "empty repo", arg: "aly.codes/", wantErr: true},
		{name: "empty", arg: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handle, repo, err := parseHandleRepo(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if handle != tt.wantHandle || repo != tt.wantRepo {
				t.Fatalf("got (%q, %q), want (%q, %q)", handle, repo, tt.wantHandle, tt.wantRepo)
			}
		})
	}
}
