package acscsemail

import "context"

type Client interface {
	SendMessage(ctx context.Context, msg AcscsMessage) error
}
