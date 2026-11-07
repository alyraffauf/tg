package app

import "context"

// EditIssue edits a ticket's title, body, or labels. Nil text fields stay unchanged.
func (s *Service) EditIssue(ctx context.Context, input EditTicketInput) (*UpdatedRecordResult, error) {
	return s.editTicket(ctx, ticketEditInput{ExpectedCID: input.ExpectedCID, Target: input.Target, Rkey: input.Rkey, Title: input.Title, Body: input.Body, Labels: input.Labels, Pull: false})
}

// EditTicketInput configures an issue or pull request edit.
type EditTicketInput struct {
	Target      Target
	Rkey        string
	Title       *string
	Body        *string
	Labels      LabelEditInput
	ExpectedCID string
}
