//go:build !windows

package fsutil

import (
	"errors"
	"os"
)

// SyncDir persists directory entries after an atomic file publication.
func SyncDir(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
