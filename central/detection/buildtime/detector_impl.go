package buildtime

import (
	"errors"
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/detection/image"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	dummyID = "dummy"
)

type detectorImpl struct {
	policySet image.PolicySet
}

// Detect runs detection on an image, returning any generated alerts.
func (d *detectorImpl) Detect(image *storage.Image) ([]*v1.Alert, error) {
	if image == nil {
		return nil, errors.New("cannot detect on a nil image")
	}
	if image.GetId() == "" {
		image.Id = dummyID
	}
	tempIndex, err := globalindex.MemOnlyIndex()
	if err != nil {
		return nil, fmt.Errorf("initializing temp index: %s", err)
	}
	tempIndexer := index.New(tempIndex)
	err = tempIndexer.AddImage(image)
	if err != nil {
		return nil, fmt.Errorf("inserting into temp index: %s", err)
	}

	var alerts []*v1.Alert
	err = d.policySet.ForEach(func(p *storage.Policy, matcher searchbasedpolicies.Matcher) error {
		violations, err := matcher.MatchOne(tempIndexer, types.NewDigest(image.GetId()).Digest())
		if err != nil {
			return fmt.Errorf("matching against policy %s: %s", p.GetName(), err)
		}
		if len(violations) > 0 {
			alerts = append(alerts, policyAndViolationsToAlert(p, violations))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

func policyAndViolationsToAlert(policy *storage.Policy, violations []*v1.Alert_Violation) *v1.Alert {
	if len(violations) == 0 {
		return nil
	}
	alert := &v1.Alert{
		Id:             uuid.NewV4().String(),
		LifecycleStage: storage.LifecycleStage_BUILD,
		Policy:         protoutils.CloneStoragePolicy(policy),
		Violations:     violations,
		Time:           ptypes.TimestampNow(),
	}
	return alert
}
