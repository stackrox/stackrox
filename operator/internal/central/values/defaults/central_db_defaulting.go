package defaults

import (
	"errors"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting"
	"k8s.io/utils/ptr"
)

var (
	CentralDBDefaultingFlow = defaulting.CentralDefaultingFlow{
		Name:           "central-db",
		DefaultingFunc: centralDBDefaulting,
	}
)

func centralDBDefaulting(logger logr.Logger, status *platform.CentralStatus, annotations map[string]string, spec *platform.CentralSpec, defaults *platform.CentralSpec) error {
	if spec == nil || spec.Central.ShouldManageDB() {
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
	} else {
		// Using an external DB (connection string supplied).

		if spec.Central != nil && spec.Central.DB.GetPersistence() != nil {
			return errors.New("if a connection string is provided, no persistence settings must be supplied")

			// TODO: there are other settings which are ignored in external mode - should we error if those are set, too?
			// Persistence seems fundamental, so it makes sense to error here, but a node selector can be regarded as more
			// accidental, that's why we tolerate it being specified. However, the reason we don't warn about it is mostly
			// that there is no good/easy way to warn.
			// Moreover, the behaviour of OpenShift console UI w.r.t. defaults is such that we cannot infer user intent
			// based merely on the (non-)nil-ness of a struct.
			// See https://github.com/stackrox/stackrox/pull/3322#discussion_r1005954280 for more details.
		}
	}
	return nil
}
