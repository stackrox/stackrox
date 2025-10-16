package common

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestCheckConnReplace(t *testing.T) {

	cases := []struct {
		a, b      *storage.SensorDeploymentIdentification
		expectErr bool
	}{
		{
			a:         nil,
			b:         nil,
			expectErr: false,
		},
		{
			a: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			b:         nil,
			expectErr: false,
		},
		{
			a: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			b: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			expectErr: false,
		},
		{
			a: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			b: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			expectErr: false,
		},
		{
			a: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			b: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			expectErr: false,
		},
		{
			a: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			b: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			expectErr: false,
		},
		{
			a: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			b: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			expectErr: false,
		},
		{
			a: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "",
			}.Build(),
			b: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			expectErr: false,
		},
		{
			a: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			b: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id2",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			expectErr: true,
		},
		{
			a: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			b: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id2",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			expectErr: true,
		},
		{
			a: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			b: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox2",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			expectErr: true,
		},
		{
			a: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			b: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id2",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			expectErr: false, // same cluster, different namespace ID
		},
		{
			a: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id",
			}.Build(),
			b: storage.SensorDeploymentIdentification_builder{
				SystemNamespaceId:   "kube-system-id",
				DefaultNamespaceId:  "default-id",
				AppNamespace:        "stackrox",
				AppNamespaceId:      "stackrox-id",
				AppServiceaccountId: "stackrox-sa-id2",
			}.Build(),
			expectErr: false, // same cluster, different service account
		},
	}

	for _, c := range cases {
		errAB := CheckConnReplace(c.a, c.b)
		errBA := CheckConnReplace(c.b, c.a)
		if c.expectErr {
			assert.Errorf(t, errAB, "expecting error when replacing connection from cluster %+v with connection from cluster %+v", c.a, c.b)
			assert.Errorf(t, errBA, "expecting error when replacing connection from cluster %+v with connection from cluster %+v", c.b, c.a)
		} else {
			assert.NoErrorf(t, errAB, "expecting no error when replacing connection from cluster %+v with connection from cluster %+v", c.a, c.b)
			assert.NoErrorf(t, errBA, "expecting no error when replacing connection from cluster %+v with connection from cluster %+v", c.b, c.a)
		}
	}
}
