package extensions

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type scannerV4StatusDefaultsReconcilliationTestCase struct {
	Spec           platform.CentralSpec
	Status         platform.CentralStatus
	ExpectedSpec   platform.CentralSpec
	ExpectedStatus platform.CentralStatus
}

var (
	errorConditions = []platform.StackRoxCondition{
		{
			Type:   platform.ConditionReleaseFailed,
			Status: platform.StatusTrue,
			Reason: platform.ReasonInstallError,
		},
	}
)

func TestReconcileScannerV4StatusDefaultsExtension(t *testing.T) {
	cases := map[string]scannerV4StatusDefaultsReconcilliationTestCase{
		"install: enabled by default": {
			Spec:   platform.CentralSpec{},
			Status: platform.CentralStatus{},
			ExpectedSpec: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Enabled,
				},
			},
			ExpectedStatus: platform.CentralStatus{
				Defaults: &platform.StatusDefaults{
					ScannerV4ComponentPolicy: string(platform.ScannerV4ComponentEnabled),
				},
			},
		},
		"install: enabled by default after incomplete install": {
			Spec: platform.CentralSpec{},
			Status: platform.CentralStatus{
				Conditions: errorConditions,
			},
			ExpectedSpec: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Enabled,
				},
			},
			ExpectedStatus: platform.CentralStatus{
				Conditions: errorConditions,
				Defaults: &platform.StatusDefaults{
					ScannerV4ComponentPolicy: string(platform.ScannerV4ComponentEnabled),
				},
			},
		},
		"upgrade: disabled by default": {
			Spec: platform.CentralSpec{},
			Status: platform.CentralStatus{
				DeployedRelease: &platform.StackRoxRelease{
					Name: "release-name",
				},
			},
			ExpectedSpec: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Disabled,
				},
			},
			ExpectedStatus: platform.CentralStatus{
				Defaults: &platform.StatusDefaults{
					ScannerV4ComponentPolicy: string(platform.ScannerV4ComponentDisabled),
				},
				DeployedRelease: &platform.StackRoxRelease{
					Name: "release-name",
				},
			},
		},
		"install: enabled explicitly": {
			Spec: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Enabled,
				},
			},
			Status: platform.CentralStatus{},
			ExpectedSpec: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Enabled,
				},
			},
			ExpectedStatus: platform.CentralStatus{},
		},
		"install: disabled explicitly": {
			Spec: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Disabled,
				},
			},
			Status: platform.CentralStatus{},
			ExpectedSpec: platform.CentralSpec{
				ScannerV4: &platform.ScannerV4Spec{
					ScannerComponent: &platform.ScannerV4Disabled,
				},
			},
			ExpectedStatus: platform.CentralStatus{},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			central := &platform.Central{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "platform.stackrox.io/v1alpha1",
					Kind:       "Central",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-central",
					Namespace: testutils.TestNamespace,
				},
				Spec:   *c.Spec.DeepCopy(),
				Status: *c.Status.DeepCopy(),
			}

			applyStatusUpdater := func(statusFunc updateStatusFunc) {
				statusFunc(&central.Status)
			}

			ctx := context.Background()
			err := reconcileScannerV4StatusDefaults(ctx, central, nil, nil, applyStatusUpdater, logr.Discard())
			assert.Nilf(t, err, "reconcileScannerV4StatusDefaults returned error: %v", err)

			assert.Equal(t, central.Spec, c.ExpectedSpec)
			assert.Equal(t, central.Status, c.ExpectedStatus)
		})
	}
}
