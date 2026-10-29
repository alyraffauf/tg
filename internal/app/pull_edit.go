package app

import "context"

func (s *Service) EditPull(ctx context.Context, t Target, rkey string, title, body *string, expectedCID ...string) (*UpdatedRecordResult, error) {
	return s.editTicket(ctx, ticketEditInput{ExpectedCID: firstCID(expectedCID), Target: t, Rkey: rkey, Title: title, Body: body, Pull: true})
}
