package imageprocessor

import (
	"fmt"
	"math"
	"strings"
	"time"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/golang/protobuf/ptypes"
)

func min(x, y float32) float32 {
	return float32(math.Min(float64(x), float64(y)))
}

func max(x, y float32) float32 {
	return float32(math.Max(float64(x), float64(y)))
}

func (rule *regexImageRule) matchComponent(image *v1.Image) (violations []*v1.Violation, ruleExists bool) {
	if rule.Component == nil {
		return
	}
	ruleExists = true
	for _, layer := range image.Scan.Layers {
		for _, component := range layer.Components {
			if rule.Component.MatchString(component.Name) {
				violation := &v1.Violation{
					Message:  fmt.Sprintf("Component '%v' matches the regex %+v", component.Name, rule.Component),
					Severity: rule.Severity,
				}
				violations = append(violations, violation)
			}
		}
	}
	return
}

func (rule *regexImageRule) matchLineRule(image *v1.Image) (violations []*v1.Violation, ruleExists bool) {
	if rule.LineRule == nil {
		return
	}
	ruleExists = true
	lineRegex := rule.LineRule
	for _, layer := range image.Metadata.Layers {
		if lineRegex.Instruction == layer.Instruction && lineRegex.Value.MatchString(layer.Value) {
			dockerFileLine := fmt.Sprintf("%v %v", layer.Instruction, layer.Value)
			violation := &v1.Violation{
				Message:  fmt.Sprintf("Dockerfile Line '%v' matches the instruction '%v' and regex '%+v'", dockerFileLine, layer.Instruction, lineRegex),
				Severity: rule.Severity,
			}
			violations = append(violations, violation)
		}
	}
	return
}

func (rule *regexImageRule) matchCVE(image *v1.Image) (violations []*v1.Violation, ruleExists bool) {
	if rule.CVE == nil {
		return
	}
	ruleExists = true
	for _, layer := range image.Scan.Layers {
		for _, component := range layer.Components {
			for _, vuln := range component.Vulns {
				if rule.CVE.MatchString(vuln.Cve) {
					violations = append(violations, &v1.Violation{
						Severity: rule.Severity,
						Message:  fmt.Sprintf("CVE '%v' matches the regex '%+v'", vuln.Cve, rule.CVE),
					})
				}
			}
		}
	}
	return
}

func (rule *regexImageRule) matchCVSS(image *v1.Image) (violations []*v1.Violation, ruleExists bool) {
	if rule.CVSS == nil {
		return
	}
	ruleExists = true
	minimum := float32(math.MaxFloat32)
	var maximum float32
	var average float32

	var numVulns float32
	for _, layer := range image.Scan.Layers {
		for _, component := range layer.Components {
			for _, vuln := range component.Vulns {
				minimum = min(minimum, vuln.Cvss)
				maximum = max(maximum, vuln.Cvss)
				average += vuln.Cvss
				numVulns++
			}
		}
	}

	var value float32
	switch rule.CVSS.MathOp {
	case v1.MathOP_MIN:
		value = minimum
	case v1.MathOP_MAX:
		value = maximum
	case v1.MathOP_AVG:
		if numVulns == 0 {
			value = 0
		} else {
			value = average / numVulns
		}
	}

	var comparatorFunc func(x, y float32) bool
	switch rule.CVSS.Op {
	case v1.Comparator_LESS_THAN:
		comparatorFunc = func(x, y float32) bool { return x < y }
	case v1.Comparator_LESS_THAN_OR_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x <= y }
	case v1.Comparator_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x == y }
	case v1.Comparator_GREATER_THAN_OR_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x >= y }
	case v1.Comparator_GREATER_THAN:
		comparatorFunc = func(x, y float32) bool { return x > y }
	}
	if comparatorFunc(value, rule.CVSS.Value) {
		violations = append(violations, &v1.Violation{
			Message:  fmt.Sprintf("CVSS component was violated"),
			Severity: rule.Severity,
		})
	}
	return
}

func (rule *regexImageRule) matchRuleToImageName(image *v1.Image) (violations []*v1.Violation, ruleExists bool) {
	if rule.ImageNameRule == nil {
		return
	}
	ruleExists = true
	if rule.ImageNameRule.Registry != nil && !rule.ImageNameRule.Registry.MatchString(image.Registry) {
		return
	}
	remoteSplit := strings.Split(image.Remote, "/")
	if len(remoteSplit) < 2 {
		// This really should never happen because image populates with defaults in the form of namespace/repo
		log.Errorf("'%v' must be of the format namespace/repo", image.Remote)
		return
	}
	namespace := remoteSplit[0]
	repo := remoteSplit[1]
	if rule.ImageNameRule.Namespace != nil && !rule.ImageNameRule.Namespace.MatchString(namespace) {
		return
	}
	if rule.ImageNameRule.Repo != nil && !rule.ImageNameRule.Repo.MatchString(repo) {
		return // return nothing if one of the regexes doesn't match. It must match all things in the image rule
	}
	if rule.ImageNameRule.Tag != nil && !rule.ImageNameRule.Tag.MatchString(image.Tag) {
		return
	}
	violations = append(violations, &v1.Violation{
		Severity: rule.Severity,
		Message:  fmt.Sprintf("Image name '%v' matches the name rule '%+v'", image.String(), *rule.ImageNameRule),
	})
	return
}

func (rule *regexImageRule) matchImageAge(image *v1.Image) (violations []*v1.Violation, ruleExists bool) {
	if rule.ImageAgeDays == 0 {
		return
	}
	ruleExists = true
	deadline := time.Now().AddDate(0, 0, -int(rule.ImageAgeDays))
	createdTime, err := ptypes.Timestamp(image.Metadata.Created)
	if err != nil {
		log.Error(err) // Log just in case, though in reality this should not occur
	}
	if createdTime.Before(deadline) {
		violations = append(violations, &v1.Violation{
			Severity: rule.Severity,
			Message:  fmt.Sprintf("Image Age '%v' is %0.2f days past the deadline", createdTime, deadline.Sub(createdTime).Hours()/24),
		})
	}
	return
}

func (rule *regexImageRule) matchScanAge(image *v1.Image) (violations []*v1.Violation, ruleExists bool) {
	if rule.ScanAgeDays == 0 {
		return
	}
	ruleExists = true
	deadline := time.Now().AddDate(0, 0, -int(rule.ScanAgeDays))
	scannedTime, err := ptypes.Timestamp(image.Scan.ScanTime)
	if err != nil {
		log.Error(err) // Log just in case, though in reality this should not occur
	}
	if scannedTime.Before(deadline) {
		violations = append(violations, &v1.Violation{
			Severity: rule.Severity,
			Message:  fmt.Sprintf("Scan Age '%v' is %0.2f days past the deadline", scannedTime, deadline.Sub(scannedTime).Hours()/24),
		})
	}
	return
}

type matchFunc func(image *v1.Image) ([]*v1.Violation, bool)

// These rules are AND'd together to make it more expressive so if any rule returns no violations
func (rule *regexImageRule) matchRuleToImage(image *v1.Image) *v1.Alert {
	matchFunctions := []matchFunc{
		rule.matchComponent,
		rule.matchLineRule,
		rule.matchCVSS,
		rule.matchRuleToImageName,
		rule.matchCVE,
		rule.matchImageAge,
		rule.matchScanAge,
	}
	var violations []*v1.Violation
	var maxSeverity v1.Severity
	// This ensures that the rule exists and if there isn't a violation of the field then it should not return any violations
	for _, f := range matchFunctions {
		calculatedViolations, exists := f(image)
		if exists && len(calculatedViolations) == 0 {
			return nil
		}
		for _, violation := range calculatedViolations {
			if violation.Severity > maxSeverity {
				maxSeverity = violation.Severity
			}
		}
		violations = append(violations, calculatedViolations...)
	}
	if len(violations) == 0 {
		return nil
	}
	alert := &v1.Alert{
		Id:         uuid.NewV4().String(),
		RuleName:   rule.Name,
		Violations: violations,
		Severity:   maxSeverity,
	}
	return alert
}
