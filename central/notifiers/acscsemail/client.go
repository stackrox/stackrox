package acscsemail

import "context"

// Client is the interface to implement for communication
// to the ACSCS email service
type Client interface {
	SendMessage(ctx context.Context, msg AcscsMessage) error
}
