package transform

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
)

var (
	_ Transformer = (*notifierTransform)(nil)

	notifierType = reflect.TypeOf((*storage.Notifier)(nil))
)

type notifierTransform struct{}

func newNotifierTransform() *notifierTransform {
	return &notifierTransform{}
}

func (r *notifierTransform) Transform(configuration declarativeconfig.Configuration) (map[reflect.Type][]proto.Message, error) {
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

	notifierProto := &storage.Notifier{
		Id:     declarativeconfig.NewDeclarativeNotifierUUID(notifierConfig.Name).String(),
		Name:   notifierConfig.Name,
		Type:   notifierTypeStr,
		Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}

	if notifierConfig.GenericConfig != nil {
		notifierProto.Config = getGenericConfig(notifierConfig.GenericConfig)
	} else if notifierConfig.SplunkConfig != nil {
		notifierProto.Config = getSplunkConfig(notifierConfig.SplunkConfig)
	} else {
		return nil, errox.InvalidArgs.Newf("unsupported notifier type %s", notifierTypeStr)
	}

	return map[reflect.Type][]proto.Message{
		notifierType: {notifierProto},
	}, nil
}

func getSplunkConfig(config *declarativeconfig.SplunkConfig) *storage.Notifier_Splunk {
	return &storage.Notifier_Splunk{
		Splunk: &storage.Splunk{
			HttpToken:           config.HTTPToken,
			HttpEndpoint:        config.HTTPEndpoint,
			Insecure:            config.Insecure,
			Truncate:            config.Truncate,
			AuditLoggingEnabled: config.AuditLoggingEnabled,
			SourceTypes:         getSourceTypes(config.SourceTypes),
		},
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
	return &storage.Notifier_Generic{
		Generic: &storage.Generic{
			Endpoint:            config.Endpoint,
			Username:            config.Username,
			Password:            config.Password,
			SkipTLSVerify:       config.SkipTLSVerify,
			CaCert:              config.CACertPEM,
			AuditLoggingEnabled: config.AuditLoggingEnabled,
			Headers:             getKeyValues(config.Headers),
			ExtraFields:         getKeyValues(config.ExtraFields),
		},
	}
}

func getKeyValues(headers []declarativeconfig.KeyValuePair) []*storage.KeyValuePair {
	res := make([]*storage.KeyValuePair, 0, len(headers))
	for _, h := range headers {
		res = append(res, &storage.KeyValuePair{
			Key:   h.Key,
			Value: h.Value,
		})
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
