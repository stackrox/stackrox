package transform

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrongConfigurationTypeTransformNotifier(t *testing.T) {
	nt := newNotifierTransform()
	msgs, err := nt.Transform(&declarativeconfig.AuthProvider{})
	assert.Nil(t, msgs)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errox.InvalidArgs)
}

func TestNoConfigTransformNotifier(t *testing.T) {
	notifier := &declarativeconfig.Notifier{
		Name: "test-notifier",
	}
	transformer := newNotifierTransform()
	protos, err := transformer.Transform(notifier)
	assert.Nil(t, protos)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errox.InvalidArgs)
}

func TestTransformGenericNotifier(t *testing.T) {
	notifier := &declarativeconfig.Notifier{
		Name: "test-notifier",
		GenericConfig: &declarativeconfig.GenericConfig{
			Endpoint:      "endpoint",
			SkipTLSVerify: true,
			CACertPEM:     "cacertpem",
			Username:      "username",
			Password:      "password",
			Headers: []declarativeconfig.KeyValuePair{
				{
					Key:   "headers-key-0",
					Value: "headers-value-0",
				},
				{
					Key:   "headers-key-1",
					Value: "headers-value-1",
				},
			},
			ExtraFields: []declarativeconfig.KeyValuePair{
				{
					Key:   "extra-fields-key-0",
					Value: "extra-fields-value-0",
				},
				{
					Key:   "extra-fields-key-1",
					Value: "extra-fields-value-1",
				},
			},
			AuditLoggingEnabled: true,
		},
	}
	expectedNotifierID := declarativeconfig.NewDeclarativeNotifierUUID(notifier.Name).String()

	transformer := newNotifierTransform()
	protos, err := transformer.Transform(notifier)
	assert.NoError(t, err)

	require.Contains(t, protos, notifierType)
	require.Len(t, protos[notifierType], 1)
	notifierProto, ok := protos[notifierType][0].(*storage.Notifier)
	require.True(t, ok)

	assert.Equal(t, storage.Traits_DECLARATIVE, notifierProto.GetTraits().GetOrigin())

	assert.Equal(t, expectedNotifierID, notifierProto.GetId())
	assert.Equal(t, notifier.Name, notifierProto.GetName())

	assert.Equal(t, "generic", notifierProto.GetType())
	assert.Nil(t, notifierProto.GetSplunk())
	assert.Equal(t, notifier.GenericConfig.Endpoint, notifierProto.GetGeneric().GetEndpoint())
	assert.Equal(t, notifier.GenericConfig.Password, notifierProto.GetGeneric().GetPassword())
	assert.Equal(t, notifier.GenericConfig.Username, notifierProto.GetGeneric().GetUsername())
	assert.Equal(t, notifier.GenericConfig.CACertPEM, notifierProto.GetGeneric().GetCaCert())
	assert.Equal(t, notifier.GenericConfig.SkipTLSVerify, notifierProto.GetGeneric().GetSkipTLSVerify())
	assert.Equal(t, notifier.GenericConfig.AuditLoggingEnabled, notifierProto.GetGeneric().GetAuditLoggingEnabled())
	assert.Len(t, notifierProto.GetGeneric().GetHeaders(), 2)
	assert.Equal(t, notifier.GenericConfig.Headers[0].Key, notifierProto.GetGeneric().GetHeaders()[0].GetKey())
	assert.Equal(t, notifier.GenericConfig.Headers[0].Value, notifierProto.GetGeneric().GetHeaders()[0].GetValue())
	assert.Equal(t, notifier.GenericConfig.Headers[1].Key, notifierProto.GetGeneric().GetHeaders()[1].GetKey())
	assert.Equal(t, notifier.GenericConfig.Headers[1].Value, notifierProto.GetGeneric().GetHeaders()[1].GetValue())
	assert.Len(t, notifierProto.GetGeneric().GetExtraFields(), 2)
	assert.Equal(t, notifier.GenericConfig.ExtraFields[0].Key, notifierProto.GetGeneric().GetExtraFields()[0].GetKey())
	assert.Equal(t, notifier.GenericConfig.ExtraFields[0].Value, notifierProto.GetGeneric().GetExtraFields()[0].GetValue())
	assert.Equal(t, notifier.GenericConfig.ExtraFields[1].Key, notifierProto.GetGeneric().GetExtraFields()[1].GetKey())
	assert.Equal(t, notifier.GenericConfig.ExtraFields[1].Value, notifierProto.GetGeneric().GetExtraFields()[1].GetValue())
}

func TestTransformSplunkNotifier(t *testing.T) {
	notifier := &declarativeconfig.Notifier{
		Name: "test-notifier",
		SplunkConfig: &declarativeconfig.SplunkConfig{
			HTTPEndpoint: "endpoint",
			Insecure:     true,
			HTTPToken:    "password",
			SourceTypes: []declarativeconfig.SourceTypePair{
				{
					Key:   "source-types-key-0",
					Value: "source-types-value-0",
				},
				{
					Key:   "source-types-key-1",
					Value: "source-types-value-1",
				},
			},
			AuditLoggingEnabled: true,
		},
	}
	expectedNotifierID := declarativeconfig.NewDeclarativeNotifierUUID(notifier.Name).String()

	transformer := newNotifierTransform()
	protos, err := transformer.Transform(notifier)
	assert.NoError(t, err)

	require.Contains(t, protos, notifierType)
	require.Len(t, protos[notifierType], 1)
	notifierProto, ok := protos[notifierType][0].(*storage.Notifier)
	require.True(t, ok)

	assert.Equal(t, storage.Traits_DECLARATIVE, notifierProto.GetTraits().GetOrigin())

	assert.Equal(t, expectedNotifierID, notifierProto.GetId())
	assert.Equal(t, notifier.Name, notifierProto.GetName())

	assert.Equal(t, "splunk", notifierProto.GetType())
	assert.Nil(t, notifierProto.GetGeneric())
	assert.Equal(t, notifier.SplunkConfig.HTTPEndpoint, notifierProto.GetSplunk().GetHttpEndpoint())
	assert.Equal(t, notifier.SplunkConfig.HTTPToken, notifierProto.GetSplunk().GetHttpToken())
	assert.Equal(t, notifier.SplunkConfig.Truncate, notifierProto.GetSplunk().GetTruncate())
	assert.Equal(t, notifier.SplunkConfig.Insecure, notifierProto.GetSplunk().GetInsecure())
	assert.Equal(t, notifier.SplunkConfig.AuditLoggingEnabled, notifierProto.GetSplunk().GetAuditLoggingEnabled())

	assert.Len(t, notifierProto.GetSplunk().GetSourceTypes(), 2)
	v, ok := notifierProto.GetSplunk().GetSourceTypes()[notifier.SplunkConfig.SourceTypes[0].Key]
	assert.True(t, ok)
	assert.Equal(t, notifier.SplunkConfig.SourceTypes[0].Value, v)
	v, ok = notifierProto.GetSplunk().GetSourceTypes()[notifier.SplunkConfig.SourceTypes[1].Key]
	assert.True(t, ok)
	assert.Equal(t, notifier.SplunkConfig.SourceTypes[1].Value, v)
}
