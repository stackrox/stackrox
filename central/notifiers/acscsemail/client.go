package acscsemail

import (
	"context"

	"github.com/stackrox/rox/central/notifiers/acscsemail/message"
)

// Client is the interface to implement for communication
// to the ACSCS email service
//
//go:generate mockgen-wrapper
type Client interface {
	SendMessage(ctx context.Context, msg message.AcscsEmail) error
}
