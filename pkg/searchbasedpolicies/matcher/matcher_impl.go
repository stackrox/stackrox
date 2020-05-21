package matcher

import (
	"context"
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/builders"
)

type matcherImpl struct {
	q *v1.Query

	processPredicate    predicate.Predicate
	deploymentPredicate predicate.Predicate
	imagePredicate      predicate.Predicate

	policyName       string
	violationPrinter searchbasedpolicies.ViolationPrinter
}

func (m *matcherImpl) errorPrefixForMatchOne() string {
	return fmt.Sprintf("matching policy %s", m.policyName)
}

// MatchOne returns detection against the deployment and images using predicate matching
// The deployment parameter can be nil in the case of image detection
func (m *matcherImpl) MatchOne(ctx context.Context, deployment *storage.Deployment, images []*storage.Image, indicator *storage.ProcessIndicator) (violations searchbasedpolicies.Violations, err error) {
	var results []*search.Result
	if indicator != nil {
		result, matches := m.processPredicate.Evaluate(indicator)
		if !matches {
			return
		}
		results = append(results, result)
	}

	if deployment != nil {
		result, matches := m.deploymentPredicate.Evaluate(deployment)
		if !matches {
			return
		}
		results = append(results, result)
	}

	if len(images) > 0 {
		var foundMatch bool
		for _, img := range images {
			result, matches := m.imagePredicate.Evaluate(img)
			if matches {
				foundMatch = true
				results = append(results, result)
			}
		}
		if !foundMatch {
			return
		}
	}

	finalResult := predicate.MergeResults(results...)
	violations = m.violationPrinter(ctx, *finalResult)
	if indicator != nil {
		v := &storage.Alert_ProcessViolation{Processes: []*storage.ProcessIndicator{indicator}}
		builders.UpdateRuntimeAlertViolationMessage(v)
		violations.ProcessViolation = v
	}
	if violationsEmpty(violations) {
		err = fmt.Errorf("%s: result matched query but couldn't find any violation messages: %+v", m.errorPrefixForMatchOne(), finalResult)
		return
	}
	return violations, nil
}

func violationsEmpty(violations searchbasedpolicies.Violations) bool {
	return len(violations.AlertViolations) == 0 && violations.ProcessViolation == nil
}
