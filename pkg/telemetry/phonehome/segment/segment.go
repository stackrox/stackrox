package segment

import (
	"fmt"
	"time"

	"github.com/gohugoio/hashstructure"
	segment "github.com/segmentio/analytics-go/v3"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

var (
	log                     = logging.LoggerForModule()
	_   telemeter.Telemeter = (*segmentTelemeter)(nil)
	// expiringIDCache stores the computed message IDs to drop duplicates if
	// requested.
	expiringIDCache = expiringcache.NewExpiringCache[string, bool](24*time.Hour, expiringcache.UpdateExpirationOnGets[string, bool])
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
func NewTelemeter(key, endpoint, clientID, clientType, clientVersion string, interval time.Duration, batchSize int) *segmentTelemeter {
	segmentConfig := segment.Config{
		Endpoint:  endpoint,
		Interval:  interval,
		BatchSize: batchSize,
		Transport: proxy.RoundTripper(),
		Logger:    &logWrapper{internal: log},
		Callback:  &logOnFailure{},
		DefaultContext: &segment.Context{
			// Client specific data, which can be overridden with WithClient:
			Device: segment.DeviceInfo{
				Id:      clientID,
				Type:    clientType,
				Version: clientVersion,
			},
			// Static data of the actual sender:
			App: segment.AppInfo{
				Version: clientVersion,
				Build:   buildinfo.BuildFlavor,
			},
			UserAgent: clientconn.GetUserAgent(),
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

// makeMessageID generates and ID based on the provided event data.
// This may allow Segment to deduplicate events.
func (t *segmentTelemeter) makeMessageID(event string, props map[string]any, o *telemeter.CallOptions) string {
	if o == nil || len(o.MessageIDPrefix) == 0 {
		return ""
	}
	h, err := hashstructure.Hash([]any{props, o.Traits, event, t.getUserID(o), t.getAnonymousID(o)}, nil)
	if err != nil {
		log.Error("Failed to generate Segment message ID: ", err)
		// Let Segment generate the id.
		return ""
	}
	return fmt.Sprintf("%s-%x", o.MessageIDPrefix, h)
}

// isDuplicate returns whether the ID exists in the cache. Adds it if not found.
func isDuplicate(id string) bool {
	if id == "" {
		return false
	}
	if _, ok := expiringIDCache.Get(id); !ok {
		expiringIDCache.Add(id, true)
		return false
	}
	return true
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
			Id:      o.ClientID,
			Type:    o.ClientType,
			Version: o.ClientVersion,
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

func (t *segmentTelemeter) prepare(event string, props map[string]any, opts []telemeter.Option) (*telemeter.CallOptions, string) {
	if t == nil {
		return nil, ""
	}
	options := telemeter.ApplyOptions(opts)
	id := t.makeMessageID(event, props, options)
	if isDuplicate(id) {
		return nil, ""
	}
	return options, id
}

func (t *segmentTelemeter) Identify(props map[string]any, opts ...telemeter.Option) {
	options, id := t.prepare("identify", props, opts)
	if options == nil {
		return
	}

	identity := segment.Identify{
		MessageId:   id,
		UserId:      t.getUserID(options),
		AnonymousId: t.getAnonymousID(options),
		Traits:      props,
		Context:     t.makeContext(options),
	}

	if err := t.client.Enqueue(identity); err != nil {
		log.Error("Cannot enqueue Segment identity event: ", err)
	}
}

func (t *segmentTelemeter) Group(props map[string]any, opts ...telemeter.Option) {
	options, id := t.prepare("group", props, opts)
	if options == nil {
		return
	}
	t.group(id, props, options)

	if len(props) != 0 {
		go func() {
			ti := time.NewTicker(2 * time.Second)
			t.groupFix(options, ti)
			ti.Stop()
		}()
	}
}

func (t *segmentTelemeter) group(id string, props map[string]any, options *telemeter.CallOptions) {
	group := segment.Group{
		MessageId:   id,
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
	options, id := t.prepare(event, props, opts)
	if options == nil {
		return
	}

	track := segment.Track{
		MessageId:   id,
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
