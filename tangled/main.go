package tangled

import (
	"log/slog"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

// Tangled is a client for the read-only bobbin XRPC API at api.tangled.org.
type Tangled struct {
	Client *atclient.APIClient
	Logger *slog.Logger
}
