package app

import (
	"errors"
	"fmt"
)

const (
	pullStateOpen      = "open"
	pullStateClosed    = "closed"
	pullStateCompleted = "completed"
)

// errPullAlreadyMerged reports an attempt to change a pull request whose
// ticket records a merge. The merge cannot be undone, so the record must not
// claim otherwise.
var errPullAlreadyMerged = errors.New("pull request is recorded as merged; its state cannot change")

// checkPullTransition enforces the pull request lifecycle on the authoritative
// ticket state: open and closed may alternate, only an open pull request can
// become completed, and completed is terminal.
func checkPullTransition(current, next string) error {
	if current == pullStateCompleted {
		return errPullAlreadyMerged
	}
	switch next {
	case pullStateOpen, pullStateClosed:
		return nil
	case pullStateCompleted:
		if current != pullStateOpen {
			return fmt.Errorf("only an open pull request can be recorded as merged; it is %q", current)
		}
		return nil
	default:
		return fmt.Errorf("unsupported pull request state %q", next)
	}
}
