package cli

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigSearchDirs(t *testing.T) {
	xdg := t.TempDir()
	home := t.TempDir()

	tests := []struct {
		name string
		xdg  string
		want []string
	}{
		{
			name: "xdg config home set",
			xdg:  xdg,
			want: []string{filepath.Join(xdg, "tg")},
		},
		{
			name: "fall back to home",
			xdg:  "",
			want: []string{filepath.Join(home, ".config", "tg")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", tt.xdg)
			t.Setenv("HOME", home)

			if got := configSearchDirs(); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadConfigKnotPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		env      string
		wantKnot string
	}{
		{name: "unset enables discovery", wantKnot: ""},
		{name: "config", config: "config.example", wantKnot: "config.example"},
		{name: "environment over config", config: "config.example", env: "env.example", wantKnot: "env.example"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xdg := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdg)
			t.Setenv("TG_KNOT", tt.env)
			if tt.config != "" {
				configDir := filepath.Join(xdg, "tg")
				if err := os.MkdirAll(configDir, 0o700); err != nil {
					t.Fatalf("create config directory: %v", err)
				}
				if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("knot = \""+tt.config+"\"\n"), 0o600); err != nil {
					t.Fatalf("write config: %v", err)
				}
			}

			got := loadConfig(flagSettings{}, io.Discard)
			if got.Knot != tt.wantKnot {
				t.Fatalf("Knot = %q, want %q", got.Knot, tt.wantKnot)
			}
		})
	}
}

func TestLoadConfigSSHPortPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		env      string
		wantPort string
	}{
		{name: "default", wantPort: "22"},
		{name: "config", config: "ssh-port = 2200\n", wantPort: "2200"},
		{name: "environment over config", config: "ssh-port = 2200\n", env: "2222", wantPort: "2222"},
		{name: "malformed config is preserved", config: "ssh-port = true\n", wantPort: "true"},
		{name: "malformed environment is preserved", env: "not-a-port", wantPort: "not-a-port"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xdg := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdg)
			t.Setenv("TG_SSH_PORT", tt.env)
			if tt.config != "" {
				configDir := filepath.Join(xdg, "tg")
				if err := os.MkdirAll(configDir, 0o700); err != nil {
					t.Fatalf("create config directory: %v", err)
				}
				if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(tt.config), 0o600); err != nil {
					t.Fatalf("write config: %v", err)
				}
			}

			got := loadConfig(flagSettings{}, io.Discard)
			if got.SSHPort != tt.wantPort {
				t.Fatalf("SSH port = %q, want %q", got.SSHPort, tt.wantPort)
			}
		})
	}
}
