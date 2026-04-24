package lane

import (
	"fmt"

	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/consumer"
)

// Type represents the type of lane implementation.
type Type string

const (
	TypeBlocking   Type = "blocking"
	TypeConcurrent Type = "concurrent"
)

// Spec defines configuration for creating a lane.
type Spec struct {
	ID       pubsub.LaneID
	Type     Type
	Size     *int // channel size (optional)
	Consumer *consumer.Spec
}

// getNewConsumer returns the NewConsumer function from the ConsumerSpec if present.
// Returns nil if no consumer spec is configured.
func (s *Spec) getNewConsumer() (pubsub.NewConsumer, error) {
	if s.Consumer == nil {
		return nil, nil
	}
	newConsumer, err := s.Consumer.ToNewConsumer()
	if err != nil {
		return nil, fmt.Errorf("invalid consumer spec for lane %s: %v", s.ID.String(), err)
	}
	return newConsumer, nil
}

// ToConfig converts the Spec to a pubsub.LaneConfig.
func (s *Spec) ToConfig() (pubsub.LaneConfig, error) {
	switch s.Type {
	case TypeBlocking:
		var opts []pubsub.LaneOption[*blockingLane]
		if s.Size != nil {
			opts = append(opts, WithBlockingLaneSize(*s.Size))
		}
		if newConsumer, err := s.getNewConsumer(); err != nil {
			return nil, err
		} else if newConsumer != nil {
			opts = append(opts, WithBlockingLaneConsumer(newConsumer))
		}
		return NewBlockingLane(s.ID, opts...), nil
	case TypeConcurrent:
		var opts []pubsub.LaneOption[*concurrentLane]
		if s.Size != nil {
			opts = append(opts, WithConcurrentLaneSize(*s.Size))
		}
		if newConsumer, err := s.getNewConsumer(); err != nil {
			return nil, err
		} else if newConsumer != nil {
			opts = append(opts, WithConcurrentLaneConsumer(newConsumer))
		}
		return NewConcurrentLane(s.ID, opts...), nil
	default:
		return nil, fmt.Errorf("unknown lane type: %s", s.Type)
	}
}

// SpecsToConfigs converts a slice of Specs to a slice of LaneConfigs.
// Returns an error if any spec is invalid.
func SpecsToConfigs(specs []Spec) ([]pubsub.LaneConfig, error) {
	configs := make([]pubsub.LaneConfig, len(specs))
	for i, spec := range specs {
		config, err := spec.ToConfig()
		if err != nil {
			return nil, fmt.Errorf("invalid lane spec at index %d: %v", i, err)
		}
		configs[i] = config
	}
	return configs, nil
}
