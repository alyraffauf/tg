package cli

import (
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
			want: []string{filepath.Join(xdg, "tg"), "."},
		},
		{
			name: "fall back to home",
			xdg:  "",
			want: []string{filepath.Join(home, ".config", "tg"), "."},
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
