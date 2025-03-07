package enrichment

import (
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/openshift"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestShouldUpsert(t *testing.T) {
	iis := []*storage.ImageIntegration{
		{Id: "Manual-GlobalPull", Autogenerated: false, Source: &storage.ImageIntegration_Source{Namespace: openshift.GlobalPullSecretNamespace, ImagePullSecretName: openshift.GlobalPullSecretName}},
		{Id: "Manual-NoSource", Autogenerated: false, Source: nil},
		{Id: "Manual-Source", Autogenerated: false, Source: &storage.ImageIntegration_Source{}},
		{Id: "Autogen-GlobalPull", Autogenerated: true, Source: &storage.ImageIntegration_Source{Namespace: openshift.GlobalPullSecretNamespace, ImagePullSecretName: openshift.GlobalPullSecretName}},
		{Id: "Autogen-NoSource", Autogenerated: true, Source: nil},
		{Id: "Autogen-Source", Autogenerated: true, Source: &storage.ImageIntegration_Source{}},
	}

	tcs := []struct {
		name                     string
		expectedUpserts          set.StringSet
		sourcedAutogenEnabled    bool
		globalPullAutogenEnabled bool
	}{
		{
			"always upsert manual integrations and those with no source",
			set.NewSet("Manual-GlobalPull", "Manual-NoSource", "Manual-Source", "Autogen-NoSource"),
			false, false,
		},
		{
			"upsert all integrations when sourced feature enabled",
			set.NewSet("Manual-GlobalPull", "Manual-NoSource", "Manual-Source", "Autogen-GlobalPull", "Autogen-NoSource", "Autogen-Source"),
			true, false,
		},
		{
			"upsert all integrations when sourced and global pull feature enabled",
			set.NewSet("Manual-GlobalPull", "Manual-NoSource", "Manual-Source", "Autogen-GlobalPull", "Autogen-NoSource", "Autogen-Source"),
			true, true,
		},
		{
			"upsert manual integrations and global pull sec integrations when only global pull feature enabled",
			set.NewSet("Manual-GlobalPull", "Manual-NoSource", "Manual-Source", "Autogen-GlobalPull", "Autogen-NoSource"),
			false, true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.SourcedAutogeneratedIntegrations, tc.sourcedAutogenEnabled)
			t.Setenv(env.AutogenerateGlobalPullSecRegistries.EnvVar(), strconv.FormatBool(tc.globalPullAutogenEnabled))

			for _, ii := range iis {
				expected := tc.expectedUpserts.Contains(ii.GetId())
				assert.Equal(t, expected, shouldUpsert(ii), "%q", ii.GetId())
			}
		})
	}
}
