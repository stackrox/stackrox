package m2m

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

func Test_kubeClaimExtractor(t *testing.T) {
	t.Run("Kubernetes Token error", func(t *testing.T) {
		e := newClaimExtractorFromConfig(&storage.AuthMachineToMachineConfig{
			Id:   "id1",
			Type: storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
		})
		testErr := errox.NotImplemented
		_, err := e.ExtractClaims(&IDToken{
			Claims: func(a any) error {
				return testErr
			},
		})
		assert.ErrorIs(t, err, errox.NotImplemented)
		assert.Contains(t, err.Error(), "extracting claims")
	})
}
