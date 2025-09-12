package defaults

import (
	"errors"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting"
	"k8s.io/utils/ptr"
)

var (
	CentralDBPersistenceDefaultingFlow = defaulting.CentralDefaultingFlow{
		Name:           "central-db",
		DefaultingFunc: centralDBPersistenceDefaulting,
	}
)

func centralDBPersistenceDefaulting(_ logr.Logger, _ *platform.CentralStatus, _ map[string]string, spec *platform.CentralSpec, defaults *platform.CentralSpec) error {
	if externalCentralDBUseSpecified(spec) {
		if centralDBPersistenceSpecified(spec) {
			return errors.New("if a connection string is provided, no persistence settings must be supplied")
		}

		// TODO: there are other settings which are ignored in external mode - should we error if those are set, too?
		// Persistence seems fundamental, so it makes sense to error here, but a node selector can be regarded as more
		// accidental, that's why we tolerate it being specified. However, the reason we don't warn about it is mostly
		// that there is no good/easy way to warn.
		// Moreover, the behaviour of OpenShift console UI w.r.t. defaults is such that we cannot infer user intent
		// based merely on the (non-)nil-ness of a struct.
		// See https://github.com/stackrox/stackrox/pull/3322#discussion_r1005954280 for more details.
		// Even though we do not use the CRD-based defaults as of 4.9, there are still central CRs in existence which
		// had been created when we did use them, so there is still no way to infer the user intent for old fields.

		return nil
	}

	return defaults.Apply(platform.CentralSpec{
		Central: &platform.CentralComponentSpec{
			DB: &platform.CentralDBSpec{
				Persistence: &platform.DBPersistence{
					PersistentVolumeClaim: &platform.DBPersistentVolumeClaim{
						ClaimName: ptr.To("central-db"),
					},
				},
			},
		},
	})
}

func externalCentralDBUseSpecified(spec *platform.CentralSpec) bool {
	return spec != nil && !spec.Central.ShouldManageDB()
}

func centralDBPersistenceSpecified(spec *platform.CentralSpec) bool {
	return spec != nil && spec.Central != nil && spec.Central.DB.GetPersistence() != nil
}
