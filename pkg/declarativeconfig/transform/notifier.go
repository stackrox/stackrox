package transform

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
	"google.golang.org/protobuf/proto"
)

var (
	_ Transformer = (*notifierTransform)(nil)

	notifierType = reflect.TypeOf((*storage.Notifier)(nil))
)

type notifierTransform struct{}

func newNotifierTransform() *notifierTransform {
	return &notifierTransform{}
}

func (r *notifierTransform) Transform(configuration declarativeconfig.Configuration) (map[reflect.Type][]protocompat.Message, error) {
	notifierConfig, ok := configuration.(*declarativeconfig.Notifier)
	if !ok {
		return nil, errox.InvalidArgs.Newf("invalid configuration type received for notifier: %T", configuration)
	}

	if notifierConfig.Name == "" {
		return nil, errox.InvalidArgs.CausedBy("name must be non-empty")
	}

	notifierTypeStr, err := getNotifierType(notifierConfig)
	if err != nil {
		return nil, errors.Wrap(err, "invalid notifier type")
	}

	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DECLARATIVE)
	notifierProto := &storage.Notifier{}
	notifierProto.SetId(declarativeconfig.NewDeclarativeNotifierUUID(notifierConfig.Name).String())
	notifierProto.SetName(notifierConfig.Name)
	notifierProto.SetType(notifierTypeStr)
	notifierProto.SetTraits(traits)

	if notifierConfig.GenericConfig != nil {
		notifierProto.SetGeneric(proto.ValueOrDefault(getGenericConfig(notifierConfig.GenericConfig).Generic))
	} else if notifierConfig.SplunkConfig != nil {
		notifierProto.SetSplunk(proto.ValueOrDefault(getSplunkConfig(notifierConfig.SplunkConfig).Splunk))
	} else {
		return nil, errox.InvalidArgs.Newf("unsupported notifier type %s", notifierTypeStr)
	}

	return map[reflect.Type][]protocompat.Message{
		notifierType: {notifierProto},
	}, nil
}

func getSplunkConfig(config *declarativeconfig.SplunkConfig) *storage.Notifier_Splunk {
	splunk := &storage.Splunk{}
	splunk.SetHttpToken(config.HTTPToken)
	splunk.SetHttpEndpoint(config.HTTPEndpoint)
	splunk.SetInsecure(config.Insecure)
	splunk.SetTruncate(config.Truncate)
	splunk.SetAuditLoggingEnabled(config.AuditLoggingEnabled)
	splunk.SetSourceTypes(getSourceTypes(config.SourceTypes))
	return &storage.Notifier_Splunk{
		Splunk: splunk,
	}
}

func getSourceTypes(types []declarativeconfig.SourceTypePair) map[string]string {
	res := map[string]string{}
	for _, t := range types {
		res[t.Key] = t.Value
	}
	return res
}

func getGenericConfig(config *declarativeconfig.GenericConfig) *storage.Notifier_Generic {
	generic := &storage.Generic{}
	generic.SetEndpoint(config.Endpoint)
	generic.SetUsername(config.Username)
	generic.SetPassword(config.Password)
	generic.SetSkipTLSVerify(config.SkipTLSVerify)
	generic.SetCaCert(config.CACertPEM)
	generic.SetAuditLoggingEnabled(config.AuditLoggingEnabled)
	generic.SetHeaders(getKeyValues(config.Headers))
	generic.SetExtraFields(getKeyValues(config.ExtraFields))
	return &storage.Notifier_Generic{
		Generic: generic,
	}
}

func getKeyValues(headers []declarativeconfig.KeyValuePair) []*storage.KeyValuePair {
	res := make([]*storage.KeyValuePair, 0, len(headers))
	for _, h := range headers {
		kvp := &storage.KeyValuePair{}
		kvp.SetKey(h.Key)
		kvp.SetValue(h.Value)
		res = append(res, kvp)
	}
	return res
}

func getNotifierType(config *declarativeconfig.Notifier) (string, error) {
	switch {
	case config.GenericConfig != nil:
		return "generic", nil
	case config.SplunkConfig != nil:
		return "splunk", nil
	default:
		return "", errox.InvalidArgs.New("no valid notifier config given")
	}
}
