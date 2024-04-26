package acscsemail

import (
	"context"
	"errors"
)

type clientImpl struct{}

var _ Client = &clientImpl{}

var client *clientImpl

func ClientSingleton() Client {
	if client != nil {
		return client
	}

	client = &clientImpl{}
	return client
}

func (s *clientImpl) SendMessage(ctx context.Context, msg AcscsMessage) error {
	return errors.New("TODO: not yet implemented")
}
