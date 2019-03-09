package buildtime

import (
	"errors"
	"fmt"
	"strings"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/detection/image"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/searchbasedpolicies"
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

func matchesImageWhitelist(image string, whitelists []*storage.Whitelist) bool {
	for _, w := range whitelists {
		if w.GetImage() == nil {
			continue
		}
		// The rationale for using a prefix is that it is the easiet way in the current format
		// to support whitelisting registries, registry/remote, etc
		if strings.HasPrefix(image, w.GetImage().GetName()) {
			return true
		}
	}
	return false
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
		return nil, fmt.Errorf("initializing temp index: %s", err)
	}
	tempIndexer := index.New(tempIndex)
	err = tempIndexer.AddImage(image)
	if err != nil {
		return nil, fmt.Errorf("inserting into temp index: %s", err)
	}

	var alerts []*storage.Alert
	err = d.policySet.ForEach(func(p *storage.Policy, matcher searchbasedpolicies.Matcher) error {
		if p.GetDisabled() {
			return nil
		}
		if matchesImageWhitelist(image.GetName().GetFullName(), p.GetWhitelists()) {
			return nil
		}
		violations, err := matcher.MatchOne(tempIndexer, types.NewDigest(image.GetId()).Digest())
		if err != nil {
			return fmt.Errorf("matching against policy %s: %s", p.GetName(), err)
		}
		alertViolations := violations.AlertViolations
		if len(alertViolations) > 0 {
			alerts = append(alerts, policyAndViolationsToAlert(p, alertViolations))
		}
		return nil
	})
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
