package tangled

import (
	"log/slog"

	"github.com/bluesky-social/indigo/xrpc"
)

// Tangled is a client for the read-only bobbin XRPC API at api.tangled.org.
type Tangled struct {
	Client *xrpc.Client
	Logger *slog.Logger
}
