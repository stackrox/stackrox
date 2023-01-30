package segment

import (
	"time"

	segment "github.com/segmentio/analytics-go/v3"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type segmentTelemeter struct {
	client segment.Client
	userID string
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

	return &segmentTelemeter{client: client, userID: clientID}
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

func (t *segmentTelemeter) overwriteID(id string) string {
	if id == "" {
		return t.userID
	}
	return id
}

func makeDeviceContext(clientID, clientType string) *segment.Context {
	if clientID == "" {
		return nil
	}
	return &segment.Context{
		Device: segment.DeviceInfo{
			Id:   clientID,
			Type: clientType,
		},
	}
}

func (t *segmentTelemeter) Identify(props map[string]any) {
	t.IdentifyUserAs("", "", "", props)
}

func (t *segmentTelemeter) IdentifyUserAs(userID, clientID, clientType string, props map[string]any) {
	if t == nil {
		return
	}
	traits := segment.NewTraits()

	identity := segment.Identify{
		UserId:  t.overwriteID(userID),
		Traits:  traits,
		Context: makeDeviceContext(clientID, clientType),
	}

	for k, v := range props {
		traits.Set(k, v)
	}
	if err := t.client.Enqueue(identity); err != nil {
		log.Error("Cannot enqueue Segment identity event: ", err)
	}
}

func (t *segmentTelemeter) Group(groupID string, props map[string]any) {
	t.GroupUserAs("", "", "", groupID, props)
}

func (t *segmentTelemeter) GroupUserAs(userID, clientID, clientType, groupID string, props map[string]any) {
	if t == nil {
		return
	}

	group := segment.Group{
		GroupId: groupID,
		UserId:  t.overwriteID(userID),
		Traits:  props,
		Context: makeDeviceContext(clientID, clientType),
	}

	if err := t.client.Enqueue(group); err != nil {
		log.Error("Cannot enqueue Segment group event: ", err)
	}
}

func (t *segmentTelemeter) Track(event string, props map[string]any) {
	t.TrackUserAs("", "", "", event, props)
}

func (t *segmentTelemeter) TrackUserAs(userID, clientID, clientType, event string, props map[string]any) {
	if t == nil {
		return
	}

	track := segment.Track{
		UserId:     t.overwriteID(userID),
		Event:      event,
		Properties: props,
		Context:    makeDeviceContext(clientID, clientType),
	}

	if err := t.client.Enqueue(track); err != nil {
		log.Error("Cannot enqueue Segment track event: ", err)
	}
}
