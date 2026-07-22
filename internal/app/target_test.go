package app

import "testing"

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name       string
		arg        string
		wantHandle string
		wantRepo   string
		wantErr    bool
	}{
		{name: "handle and repo", arg: "aly.codes/tg", wantHandle: "aly.codes", wantRepo: "tg"},
		{name: "did handle", arg: "did:plc:abc123/tg", wantHandle: "did:plc:abc123", wantRepo: "tg"},
		// Repo names become atproto record keys, which cannot contain "/".
		{name: "repo containing slash", arg: "aly.codes/a/b", wantErr: true},
		{name: "no slash", arg: "tg", wantErr: true},
		{name: "empty handle", arg: "/tg", wantErr: true},
		{name: "empty repo", arg: "aly.codes/", wantErr: true},
		{name: "empty", arg: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := ParseTarget(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if target.Handle != tt.wantHandle || target.Repo != tt.wantRepo {
				t.Fatalf("got (%q, %q), want (%q, %q)", target.Handle, target.Repo, tt.wantHandle, tt.wantRepo)
			}
		})
	}
}

func TestTargetString(t *testing.T) {
	target := Target{Handle: "aly.codes", Repo: "tg"}
	if got := target.String(); got != "aly.codes/tg" {
		t.Fatalf("String() = %q, want %q", got, "aly.codes/tg")
	}
}
