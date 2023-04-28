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
	client     segment.Client
	clientID   string
	clientType string
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

func (*logOnFailure) Success(_ segment.Message) {}
func (*logOnFailure) Failure(msg segment.Message, err error) {
	log.Error("Failure with message '", getMessageType(msg), "': ", err)
}

// NewTelemeter creates and initializes a Segment telemeter instance.
// Default interval is 5s, default batch size is 250.
func NewTelemeter(key, endpoint, clientID, clientType string, interval time.Duration, batchSize int) *segmentTelemeter {
	segmentConfig := segment.Config{
		Endpoint:  endpoint,
		Interval:  interval,
		BatchSize: batchSize,
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

	return &segmentTelemeter{client: client, clientID: clientID, clientType: clientType}
}

type logWrapper struct {
	internal logging.Logger
}

func (l *logWrapper) Logf(format string, args ...any) {
	l.internal.Infof(format, args...)
}

func (l *logWrapper) Errorf(format string, args ...any) {
	l.internal.Errorf(format, args...)
}

func (t *segmentTelemeter) Stop() {
	if t == nil {
		return
	}
	if err := t.client.Close(); err != nil {
		log.Error("Cannot close Segment client: ", err)
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

func (t *segmentTelemeter) makeContext(o *telemeter.CallOptions) *segment.Context {
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

	if o.UserID == "" {
		// Add "Server" suffix to the platform of the backend initiated events:
		if ctx == nil {
			ctx = &segment.Context{}
		}
		if ctx.Device.Type == "" {
			ctx.Device.Type = t.clientType
		}
		ctx.Device.Type += " Server"
	}

	if o.Traits != nil {
		if ctx == nil {
			ctx = &segment.Context{}
		}
		ctx.Traits = o.Traits
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
		Context:     t.makeContext(options),
	}

	for k, v := range props {
		traits.Set(k, v)
	}
	if err := t.client.Enqueue(identity); err != nil {
		log.Error("Cannot enqueue Segment identity event: ", err)
	}
}

func (t *segmentTelemeter) Group(props map[string]any, opts ...telemeter.Option) {
	if t == nil {
		return
	}
	options := telemeter.ApplyOptions(opts)
	t.group(props, options)

	if len(props) != 0 {
		go func() {
			ti := time.NewTicker(2 * time.Second)
			t.groupFix(options, ti)
			ti.Stop()
		}()
	}
}

func (t *segmentTelemeter) group(props map[string]any, options *telemeter.CallOptions) {
	group := segment.Group{
		UserId:      t.getUserID(options),
		AnonymousId: t.getAnonymousID(options),
		Traits:      props,
		Context:     t.makeContext(options),
	}

	for _, ids := range options.Groups {
		if len(ids) == 0 {
			continue
		}

		// Segment doesn't understand group Type. The type must be configured
		// in the Amplitude destination mapping.
		group.GroupId = ids[0]

		if err := t.client.Enqueue(group); err != nil {
			log.Error("Cannot enqueue Segment group event: ", err)
		}
	}
}

func (t *segmentTelemeter) groupFix(options *telemeter.CallOptions, ti *time.Ticker) {
	// Track the group properties update with the same device ID
	// to ensure following events get the properties attached. This is
	// due to Amplitude partioning by device ID.
	track := segment.Track{
		UserId:      t.getUserID(options),
		AnonymousId: t.getAnonymousID(options),
		Event:       "Group Properties Updated",
		Context:     t.makeContext(options),
	}

	// Segment does not guarantee the processing order of the events,
	// we need, therefore, to add a delay between Group and Track to
	// ensure the Track catches the group properties. We do it several
	// times to raise the chances for the potential events from other
	// clients coming in between to capture the group properties.
	for i := 0; i < 3; i++ {
		if i != 0 {
			<-ti.C
		}
		if err := t.client.Enqueue(track); err != nil {
			log.Error("Cannot enqueue Segment track event: ", err)
			break
		}
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
		Context:     t.makeContext(options),
	}

	if err := t.client.Enqueue(track); err != nil {
		log.Error("Cannot enqueue Segment track event: ", err)
	}
}
