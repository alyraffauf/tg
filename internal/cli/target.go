package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alyraffauf/tg/internal/app"
)

// getwd returns the current working directory, wrapping the common error.
func getwd() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current directory: %w", err)
	}
	return dir, nil
}

// resolveTarget returns the target from an explicit "handle/repo" argument
// (when args has one element) or by detecting the git remote in the CWD.
func resolveTarget(ctx context.Context, args []string, service *app.Service) (app.Target, error) {
	if len(args) == 1 {
		return app.ParseTarget(args[0])
	}
	return service.TargetFromCWD(ctx)
}

// resolveTargetFlag returns the target from a --repo flag value, or by
// detecting the git remote in the CWD when the flag is unset.
func resolveTargetFlag(ctx context.Context, repoFlag string, service *app.Service) (app.Target, error) {
	if repoFlag != "" {
		return app.ParseTarget(repoFlag)
	}
	return service.TargetFromCWD(ctx)
}

type accountHandleResolver interface {
	HandleOrSelf(context.Context, string) (string, error)
}

// resolveHandleOrSelf returns the handle from an explicit argument, or the
// authenticated user's handle. It does not fall back to CWD git detection.
func resolveHandleOrSelf(ctx context.Context, args []string, service accountHandleResolver) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	return service.HandleOrSelf(ctx, "")
}

// resolveCloneTarget accepts either a complete handle/repo target or a repo
// name owned by the authenticated user.
func resolveCloneTarget(ctx context.Context, arg string, service accountHandleResolver) (app.Target, error) {
	if strings.Contains(arg, "/") {
		return app.ParseTarget(arg)
	}
	if arg == "" {
		return app.Target{}, fmt.Errorf("expected repo or handle/repo, got %q", arg)
	}

	handle, err := service.HandleOrSelf(ctx, "")
	if err != nil {
		return app.Target{}, err
	}
	return app.Target{Handle: handle, Repo: arg}, nil
}
