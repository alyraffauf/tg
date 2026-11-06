package fsutil

// SyncDir is a no-op on Windows, where read-only directory handles cannot be
// flushed with FlushFileBuffers. Callers must flush file contents before publishing.
func SyncDir(string) error { return nil }
