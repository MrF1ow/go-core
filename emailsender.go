package core

import "context"

// EmailSender abstracts email transport.
// The core handles template rendering; this interface handles the actual sending.
type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody, textBody string) error
}
