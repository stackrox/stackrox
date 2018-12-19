package manager

import (
	"io"

	"github.com/stackrox/rox/generated/internalapi/central"
)

type networkFlowStream struct {
	conn *sensorConnection
}

func (c *sensorConnection) newNetworkFlowStream() *networkFlowStream {
	return &networkFlowStream{
		conn: c,
	}
}

func (c *sensorConnection) injectNetworkFlowUpdate(update *central.NetworkFlowUpdate) error {
	if err := c.stopSig.ErrorWithDefault(io.EOF); err != nil {
		return err
	}

	select {
	case c.flowUpdatesC <- update:
		return nil
	case <-c.stopSig.Done():
		return c.stopSig.Err()
	}
}

func (s *networkFlowStream) Recv() (*central.NetworkFlowUpdate, error) {
	if err := s.conn.stopSig.ErrorWithDefault(io.EOF); err != nil {
		return nil, err
	}

	select {
	case update := <-s.conn.flowUpdatesC:
		return update, nil
	case <-s.conn.stopSig.Done():
		return nil, s.conn.stopSig.Err()
	}
}
