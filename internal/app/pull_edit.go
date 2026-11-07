package app

import "context"

// EditPull edits a ticket's title, body, or labels. Nil text fields stay unchanged.
func (s *Service) EditPull(ctx context.Context, input EditTicketInput) (*UpdatedRecordResult, error) {
	return s.editTicket(ctx, ticketEditInput{ExpectedCID: input.ExpectedCID, Target: input.Target, Rkey: input.Rkey, Title: input.Title, Body: input.Body, Labels: input.Labels, Pull: true})
}
