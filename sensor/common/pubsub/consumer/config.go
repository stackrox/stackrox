package consumer

import (
	"fmt"

	"github.com/stackrox/rox/sensor/common/pubsub"
)

// Type represents the type of consumer implementation.
type Type string

const (
	TypeDefault  Type = "default"
	TypeBuffered Type = "buffered"
)

// Spec defines configuration for creating a consumer.
type Spec struct {
	Type Type
	Size *int // buffer size for buffered consumer (optional)
}

// ToNewConsumer converts the Spec to a pubsub.NewConsumer function.
func (s *Spec) ToNewConsumer() (pubsub.NewConsumer, error) {
	if s == nil {
		return NewDefaultConsumer(), nil
	}

	switch s.Type {
	case TypeDefault, "":
		return NewDefaultConsumer(), nil
	case TypeBuffered:
		var opts []pubsub.ConsumerOption[*BufferedConsumer]
		if s.Size != nil {
			opts = append(opts, WithBufferedConsumerSize(*s.Size))
		}
		return NewBufferedConsumer(opts...), nil
	default:
		return nil, fmt.Errorf("unknown consumer type: %s", s.Type)
	}
}
