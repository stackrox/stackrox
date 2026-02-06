package config

import (
	"fmt"

	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/consumer"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
)

// LaneType represents the type of lane implementation.
type LaneType string

const (
	LaneTypeBlocking   LaneType = "blocking"
	LaneTypeConcurrent LaneType = "concurrent"
)

// ConsumerType represents the type of consumer implementation.
type ConsumerType string

const (
	ConsumerTypeDefault  ConsumerType = "default"
	ConsumerTypeBuffered ConsumerType = "buffered"
)

// LaneSpec defines configuration for creating a lane.
type LaneSpec struct {
	ID       pubsub.LaneID
	Type     LaneType
	Size     *int // channel size (optional)
	Consumer *ConsumerSpec
}

// ConsumerSpec defines configuration for creating a consumer.
type ConsumerSpec struct {
	Type ConsumerType
	Size *int // buffer size for buffered consumer (optional)
}

// ToNewConsumer converts the ConsumerSpec to a pubsub.NewConsumer function.
func (s *ConsumerSpec) ToNewConsumer() (pubsub.NewConsumer, error) {
	if s == nil {
		return consumer.NewDefaultConsumer(), nil
	}

	switch s.Type {
	case ConsumerTypeDefault, "":
		return consumer.NewDefaultConsumer(), nil
	case ConsumerTypeBuffered:
		var opts []pubsub.ConsumerOption[*consumer.BufferedConsumer]
		if s.Size != nil {
			opts = append(opts, consumer.WithBufferedConsumerSize(*s.Size))
		}
		return consumer.NewBufferedConsumer(opts...), nil
	default:
		return nil, fmt.Errorf("unknown consumer type: %s", s.Type)
	}
}

// getNewConsumer returns the NewConsumer function from the ConsumerSpec if present.
// Returns nil if no consumer spec is configured.
func (s *LaneSpec) getNewConsumer() (pubsub.NewConsumer, error) {
	if s.Consumer == nil {
		return nil, nil
	}
	newConsumer, err := s.Consumer.ToNewConsumer()
	if err != nil {
		return nil, fmt.Errorf("invalid consumer spec for lane %s: %v", s.ID.String(), err)
	}
	return newConsumer, nil
}

// ToConfig converts the LaneSpec to a pubsub.LaneConfig.
func (s *LaneSpec) ToConfig() (pubsub.LaneConfig, error) {
	switch s.Type {
	case LaneTypeBlocking:
		var opts []pubsub.LaneOption[*lane.BlockingLane]
		if s.Size != nil {
			opts = append(opts, lane.WithBlockingLaneSize(*s.Size))
		}
		if newConsumer, err := s.getNewConsumer(); err != nil {
			return nil, err
		} else if newConsumer != nil {
			opts = append(opts, lane.WithBlockingLaneConsumer(newConsumer))
		}
		return lane.NewBlockingLane(s.ID, opts...), nil
	case LaneTypeConcurrent:
		var opts []pubsub.LaneOption[*lane.ConcurrentLane]
		if s.Size != nil {
			opts = append(opts, lane.WithConcurrentLaneSize(*s.Size))
		}
		if newConsumer, err := s.getNewConsumer(); err != nil {
			return nil, err
		} else if newConsumer != nil {
			opts = append(opts, lane.WithConcurrentLaneConsumer(newConsumer))
		}
		return lane.NewConcurrentLane(s.ID, opts...), nil
	default:
		return nil, fmt.Errorf("unknown lane type: %s", s.Type)
	}
}

// SpecsToConfigs converts a slice of LaneSpecs to a slice of LaneConfigs.
// Returns an error if any spec is invalid.
func SpecsToConfigs(specs []LaneSpec) ([]pubsub.LaneConfig, error) {
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
