package prometheus

import (
	"crypto/x509/pkix"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
)

func TestAuthorizedPrometheusClient(t *testing.T) {
	t.Setenv(env.SecureMetricsClientCertCN.EnvVar(), "system:serviceaccount:openshift-monitoring:prometheus-k8s")
	expectedCN := env.SecureMetricsClientCertCN.Setting()

	t.Run("allows verified prometheus client cert", func(t *testing.T) {
		assert.True(t, authorizedPrometheusClientFromSubjects(
			[]pkix.Name{{CommonName: expectedCN}},
			expectedCN,
		))
	})

	t.Run("denies missing client cert", func(t *testing.T) {
		assert.False(t, authorizedPrometheusClientFromSubjects(nil, expectedCN))
	})

	t.Run("denies unexpected client cert", func(t *testing.T) {
		assert.False(t, authorizedPrometheusClientFromSubjects(
			[]pkix.Name{{CommonName: "system:serviceaccount:default:other"}},
			expectedCN,
		))
	})
}
