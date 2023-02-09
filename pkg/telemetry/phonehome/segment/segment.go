package segment

import (
	"time"

	segment "github.com/segmentio/analytics-go/v3"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

var (
	log                     = logging.LoggerForModule()
	_   telemeter.Telemeter = (*segmentTelemeter)(nil)
)

type segmentTelemeter struct {
	client   segment.Client
	clientID string
}

func getMessageType(msg segment.Message) string {
	switch m := msg.(type) {
	case segment.Alias:
		return m.Type
	case segment.Group:
		return m.Type
	case segment.Identify:
		return m.Type
	case segment.Page:
		return m.Type
	case segment.Screen:
		return m.Type
	case segment.Track:
		return m.Type
	default:
		return ""
	}
}

type logOnFailure struct{}

func (*logOnFailure) Success(msg segment.Message) {}
func (*logOnFailure) Failure(msg segment.Message, err error) {
	log.Error("Failure with message '", getMessageType(msg), "': ", err)
}

// NewTelemeter creates and initializes a Segment telemeter instance.
func NewTelemeter(key, endpoint, clientID, clientType string, interval time.Duration) *segmentTelemeter {
	segmentConfig := segment.Config{
		Endpoint:  endpoint,
		Interval:  interval,
		Transport: proxy.RoundTripper(),
		Logger:    &logWrapper{internal: log},
		Callback:  &logOnFailure{},
		DefaultContext: &segment.Context{
			Device: segment.DeviceInfo{
				Id:   clientID,
				Type: clientType,
			},
		},
	}

	client, err := segment.NewWithConfig(key, segmentConfig)
	if err != nil {
		log.Error("Cannot initialize Segment client: ", err)
		return nil
	}

	return &segmentTelemeter{client: client, clientID: clientID}
}

type logWrapper struct {
	internal *logging.Logger
}

func (l *logWrapper) Logf(format string, args ...any) {
	l.internal.Infof(format, args...)
}

func (l *logWrapper) Errorf(format string, args ...any) {
	l.internal.Errorf(format, args...)
}

func (t *segmentTelemeter) Stop() {
	if t != nil {
		if err := t.client.Close(); err != nil {
			log.Error("Cannot close Segment client: ", err)
		}
	}
}

func (t *segmentTelemeter) getUserID(o *telemeter.CallOptions) string {
	if o.AnonymousID != "" {
		return ""
	}
	return o.UserID
}

func (t *segmentTelemeter) getAnonymousID(o *telemeter.CallOptions) string {
	if o.UserID != "" {
		return ""
	}
	if o.AnonymousID != "" {
		return o.AnonymousID
	}
	if o.ClientID != "" {
		return o.ClientID
	}
	return t.clientID
}

func makeDeviceContext(o *telemeter.CallOptions) *segment.Context {
	var ctx *segment.Context

	if len(o.Groups) > 0 {
		// Add groups to the context. Requires a mapping configuration for
		// setting the according Amplitude event field.
		ctx = &segment.Context{
			Extra: map[string]any{"groups": o.Groups},
		}
	}

	if o.ClientID != "" {
		if ctx == nil {
			ctx = &segment.Context{}
		}
		ctx.Device = segment.DeviceInfo{
			Id:   o.ClientID,
			Type: o.ClientType,
		}
	}
	return ctx
}

func (t *segmentTelemeter) Identify(props map[string]any, opts ...telemeter.Option) {
	if t == nil {
		return
	}

	options := telemeter.ApplyOptions(opts)

	traits := segment.NewTraits()

	identity := segment.Identify{
		UserId:      t.getUserID(options),
		AnonymousId: t.getAnonymousID(options),
		Traits:      traits,
		Context:     makeDeviceContext(options),
	}

	for k, v := range props {
		traits.Set(k, v)
	}
	if err := t.client.Enqueue(identity); err != nil {
		log.Error("Cannot enqueue Segment identity event: ", err)
	}
}

func (t *segmentTelemeter) Group(groupID string, props map[string]any, opts ...telemeter.Option) {
	if t == nil {
		return
	}

	options := telemeter.ApplyOptions(opts)

	group := segment.Group{
		GroupId:     groupID,
		UserId:      t.getUserID(options),
		AnonymousId: t.getAnonymousID(options),
		Traits:      props,
		Context:     makeDeviceContext(options),
	}

	if err := t.client.Enqueue(group); err != nil {
		log.Error("Cannot enqueue Segment group event: ", err)
	}
}

func (t *segmentTelemeter) Track(event string, props map[string]any, opts ...telemeter.Option) {
	if t == nil {
		return
	}

	options := telemeter.ApplyOptions(opts)

	track := segment.Track{
		UserId:      t.getUserID(options),
		AnonymousId: t.getAnonymousID(options),
		Event:       event,
		Properties:  props,
		Context:     makeDeviceContext(options),
	}

	if err := t.client.Enqueue(track); err != nil {
		log.Error("Cannot enqueue Segment track event: ", err)
	}
}
