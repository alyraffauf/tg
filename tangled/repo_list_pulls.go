package tangled

import "context"

func (t *Tangled) ListPulls(ctx context.Context, repo string, opts ListOpts) (*List, error) {
	return t.listTickets(ctx, repo, opts, true)
}
