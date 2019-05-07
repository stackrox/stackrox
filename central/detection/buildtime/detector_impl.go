package buildtime

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/detection"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	dummyID = "dummy"
)

type detectorImpl struct {
	policySet detection.PolicySet
}

// Detect runs detection on an image, returning any generated alerts.
func (d *detectorImpl) Detect(image *storage.Image) ([]*storage.Alert, error) {
	if image == nil {
		return nil, errors.New("cannot detect on a nil image")
	}
	if image.GetId() == "" {
		image.Id = dummyID
	}
	tempIndex, err := globalindex.MemOnlyIndex()
	if err != nil {
		return nil, errors.Wrap(err, "initializing temp index")
	}
	tempIndexer := index.New(tempIndex)
	err = tempIndexer.AddImage(image)
	if err != nil {
		return nil, errors.Wrap(err, "inserting into temp index")
	}
	tempSearcher := search.WrapContextLessSearcher(tempIndexer)

	var alerts []*storage.Alert
	err = d.policySet.ForEach(detection.FunctionAsExecutor(func(compiled detection.CompiledPolicy) error {
		if compiled.Policy().GetDisabled() {
			return nil
		}
		if !compiled.AppliesTo(image) {
			return nil
		}
		violations, err := compiled.Matcher().MatchOne(context.TODO(), tempSearcher, types.NewDigest(image.GetId()).Digest())
		if err != nil {
			return errors.Wrapf(err, "matching against policy %s", compiled.Policy().GetName())
		}
		alertViolations := violations.AlertViolations
		if len(alertViolations) > 0 {
			alerts = append(alerts, policyAndViolationsToAlert(compiled.Policy(), alertViolations))
		}
		return nil
	}))
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

func policyAndViolationsToAlert(policy *storage.Policy, violations []*storage.Alert_Violation) *storage.Alert {
	if len(violations) == 0 {
		return nil
	}
	alert := &storage.Alert{
		Id:             uuid.NewV4().String(),
		LifecycleStage: storage.LifecycleStage_BUILD,
		Policy:         protoutils.CloneStoragePolicy(policy),
		Violations:     violations,
		Time:           ptypes.TimestampNow(),
	}
	return alert
}
