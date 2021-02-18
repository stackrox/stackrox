package buildtime

import (
	"fmt"
	"strings"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
)

type detectorImpl struct {
	policySet detection.PolicySet
}

// Detect runs detection on an image, returning any generated alerts.  If policy categories are specified, we will only
// run policies with the specified categories
func (d *detectorImpl) Detect(image *storage.Image, policyCategories []string) ([]*storage.Alert, error) {
	if image == nil {
		return nil, errors.New("cannot detect on a nil image")
	}

	allowedCategorySet := set.NewStringSet()
	// Keep track of the categories which haven't been used yet.  We want to return an error if categories are specified
	// that don't exist.
	unusedCategorySet := set.NewStringSet()
	for _, category := range policyCategories {
		lowercaseCategory := strings.ToLower(category)
		allowedCategorySet.Add(lowercaseCategory)
		unusedCategorySet.Add(lowercaseCategory)
	}

	var alerts []*storage.Alert
	var cacheReceptacle booleanpolicy.CacheReceptacle
	err := d.policySet.ForEach(func(compiled detection.CompiledPolicy) error {
		if compiled.Policy().GetDisabled() {
			return nil
		}
		if !hasAllowedCatgories(allowedCategorySet, unusedCategorySet, compiled.Policy().Categories) {
			return nil
		}
		if !compiled.AppliesTo(image) {
			return nil
		}
		violations, err := compiled.MatchAgainstImage(&cacheReceptacle, image)
		if err != nil {
			return errors.Wrapf(err, "matching against policy %s", compiled.Policy().GetName())
		}
		alertViolations := violations.AlertViolations
		if len(alertViolations) > 0 {
			alerts = append(alerts, policyViolationsAndImageToAlert(compiled.Policy(), alertViolations, image))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if unusedCategorySet.Cardinality() > 0 {
		return nil, fmt.Errorf("allowed categories %v did not match any policy categories", unusedCategorySet.AsSlice())
	}
	return alerts, nil
}

func policyViolationsAndImageToAlert(policy *storage.Policy, violations []*storage.Alert_Violation, image *storage.Image) *storage.Alert {
	if len(violations) == 0 {
		return nil
	}
	alert := &storage.Alert{
		Id:             uuid.NewV4().String(),
		LifecycleStage: storage.LifecycleStage_BUILD,
		Policy:         policy.Clone(),
		Entity:         &storage.Alert_Image{Image: types.ToContainerImage(image)},
		Violations:     violations,
		Time:           ptypes.TimestampNow(),
	}
	return alert
}

func hasAllowedCatgories(categorySet set.StringSet, unmatchedSet set.StringSet, policyCategories []string) bool {
	// If categorySet is empty we default to all categories
	if categorySet.Cardinality() == 0 {
		return true
	}

	matched := false
	for _, category := range policyCategories {
		lowercaseCategory := strings.ToLower(category)
		if categorySet.Contains(lowercaseCategory) {
			matched = true
			unmatchedSet.Remove(lowercaseCategory)
		}
	}

	return matched
}
