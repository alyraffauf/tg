package app

import (
	"context"
	"strings"
	"testing"
)

func TestCreateRepoValidatesConnectionSettingsBeforeProvisioning(t *testing.T) {
	tests := []struct {
		name        string
		knotHost    string
		sshPort     int
		pushPath    string
		clone       bool
		wantHost    string
		wantErr     string
		wantClones  int
		wantCreates int
		wantPuts    int
	}{
		{name: "malformed hostname", knotHost: "https://knot.example/path", wantErr: "invalid Knot hostname"},
		{name: "zero push port", knotHost: "knot.example", pushPath: ".", wantErr: "SSH port"},
		{name: "negative push port", knotHost: "knot.example", sshPort: -1, pushPath: ".", wantErr: "SSH port"},
		{name: "push port above maximum", knotHost: "knot.example", sshPort: 65536, pushPath: ".", wantErr: "SSH port"},
		{name: "zero clone port", knotHost: "knot.example", clone: true, wantErr: "SSH port"},
		{name: "valid custom port", knotHost: "knot.example", sshPort: 2222, pushPath: ".", wantHost: "knot.example", wantCreates: 1, wantPuts: 1},
		{name: "clone from custom knot and port", knotHost: "knot.example", sshPort: 2222, clone: true, wantHost: "knot.example", wantClones: 1, wantCreates: 1, wantPuts: 1},
		{name: "hostname is canonicalized", knotHost: "KNOT.EXAMPLE", wantHost: "knot.example", wantCreates: 1, wantPuts: 1},
		{name: "unused invalid port", knotHost: "knot.example", wantHost: "knot.example", wantCreates: 1, wantPuts: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pds := &testPDS{}
			knotClient := &testKnot{}
			git := &testGit{}
			service := testService(pds, git, knotClient)

			result, err := service.CreateRepo(context.Background(), CreateRepoInput{
				KnotHost: tt.knotHost, SSHPort: tt.sshPort, Name: "example", Clone: tt.clone, PushPath: tt.pushPath,
			})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("CreateRepo() error = %v, want containing %q", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("CreateRepo() error = %v", err)
				}
				if result.Knot != tt.wantHost {
					t.Fatalf("CreateRepo() Knot = %q, want %q", result.Knot, tt.wantHost)
				}
			}
			if knotClient.createCalls != tt.wantCreates || len(pds.puts) != tt.wantPuts {
				t.Fatalf("create/put calls = %d/%d, want %d/%d", knotClient.createCalls, len(pds.puts), tt.wantCreates, tt.wantPuts)
			}
			if len(git.clones) != tt.wantClones {
				t.Fatalf("clone calls = %+v, want %d", git.clones, tt.wantClones)
			}
			if tt.wantClones == 1 && (git.clones[0].KnotHost != tt.wantHost || git.clones[0].SSHPort != tt.sshPort) {
				t.Fatalf("clone destination = %+v, want %s:%d", git.clones[0], tt.wantHost, tt.sshPort)
			}
			if tt.wantErr != "" && (pds.serviceAuthCalls != 0 || len(git.pushes) != 0) {
				t.Fatalf("service auth/push calls = %d/%d, want no side effects", pds.serviceAuthCalls, len(git.pushes))
			}
			if tt.wantErr == "" {
				wantAudience := "did:web:" + tt.wantHost
				if len(pds.serviceAuthAudiences) == 0 {
					t.Fatal("service auth was not requested")
				}
				for _, audience := range pds.serviceAuthAudiences {
					if audience != wantAudience {
						t.Fatalf("service auth audience = %q, want %q", audience, wantAudience)
					}
				}
			}
		})
	}
}
