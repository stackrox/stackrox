package errors

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

var (
	UndefinedEventCallbackErr = errors.New("undefined event callback")
	WaitableTriggeredErr      = errors.New("waitable was triggered")
	ChannelFullErr            = errors.New("channel is full")
)

func NewPublishOnStoppedLaneErr(id pubsub.LaneID) error {
	return errors.Errorf("publishing on stopped lane %q", id.String())
}

func NewConsumersNotFoundForTopicErr(topic pubsub.Topic, laneID pubsub.LaneID) error {
	return errors.Errorf("no consumers found in lane %q for topic %q", laneID.String(), topic.String())
}

func WrapConsumeErr(err error, topic pubsub.Topic, laneID pubsub.LaneID) error {
	return errors.Wrapf(err, "unable to consume event in lane %q for topic %q", laneID.String(), topic.String())
}

func NewConsumerBufferFullError(topic pubsub.Topic, laneID pubsub.LaneID) error {
	return errors.Errorf("the consumer buffer is full. Lane %q Topic %q", laneID.String(), topic.String())
}

func NewConsumeOnStoppedConsumerErr(topic pubsub.Topic, laneID pubsub.LaneID) error {
	return errors.Errorf("publishing on stopped consumer. Lane %q Topic %q", laneID.String(), topic.String())
}
