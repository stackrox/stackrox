package messagestream

import (
	"math"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/reflectutils"
)

type sizingEventStream struct {
	stream  CentralMessageStream
	maxSeen map[string]float64
}

func (s *sizingEventStream) Send(msg *central.MsgToSensor) error {
	typ := reflectutils.Type(msg.GetMsg())
	gaugeValue := math.Max(s.maxSeen[typ], float64(msg.SizeVT()))
	metrics.ObserveSentSize(typ, float64(msg.SizeVT()))
	metrics.SetGRPCLastMessageSizeGauge(typ, float64(msg.SizeVT()))
	metrics.SetGRPCMaxMessageSizeGauge(typ, gaugeValue)
	s.maxSeen[typ] = gaugeValue
	return s.stream.Send(msg)
}

// NewSizingEventStream returns a new CentralMessageStream that automatically updates max message size sent metric.
func NewSizingEventStream(stream CentralMessageStream) CentralMessageStream {
	return &sizingEventStream{stream, make(map[string]float64)}
}
