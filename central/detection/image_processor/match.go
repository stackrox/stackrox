package imageprocessor

import (
	"fmt"
	"math"
	"strings"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"github.com/golang/protobuf/ptypes"
)

const cveLinkPrefix = "https://nvd.nist.gov/vuln/detail/"

type matchFunc func(image *v1.Image) ([]*v1.Alert_Violation, bool)

// Match matches the policy if *ALL* conditions of the policy are satisfied.
func (policy *compiledImagePolicy) Match(deployment *v1.Deployment, container *v1.Container) (violations []*v1.Alert_Violation) {
	image := container.GetImage()
	if image == nil {
		return
	}

	matchFunctions := []matchFunc{
		policy.matchComponent,
		policy.matchLineRule,
		policy.matchCVSS,
		policy.matchImageName,
		policy.matchCVE,
		policy.matchImageAge,
		policy.matchScanAge,
		policy.matchScanExists,
	}
	// This ensures that the policy exists and if there isn't a violation of the field then it should not return any violations
	for _, f := range matchFunctions {
		calculatedViolations, exists := f(image)
		if exists && len(calculatedViolations) == 0 {
			return nil
		}
		violations = append(violations, calculatedViolations...)
	}

	return
}

func min(x, y float32) float32 {
	return float32(math.Min(float64(x), float64(y)))
}

func max(x, y float32) float32 {
	return float32(math.Max(float64(x), float64(y)))
}

func (policy *compiledImagePolicy) matchComponent(image *v1.Image) (violations []*v1.Alert_Violation, policyExists bool) {
	if policy.Component == nil {
		return
	}
	policyExists = true
	for _, component := range image.GetScan().GetComponents() {
		if policy.Component.Name.MatchString(component.GetName()) && policy.Component.Version.MatchString(component.GetVersion()) {
			violation := &v1.Alert_Violation{
				Message: fmt.Sprintf("Component '%v:%v' matches name regex %+v and value regex %+v", component.GetName(), component.GetVersion(), policy.Component.Name, policy.Component.Version),
			}
			violations = append(violations, violation)
		}
	}
	return
}

func (policy *compiledImagePolicy) matchLineRule(image *v1.Image) (violations []*v1.Alert_Violation, policyExists bool) {
	if policy.LineRule == nil {
		return
	}
	policyExists = true
	lineRegex := policy.LineRule
	for _, layer := range image.GetMetadata().GetLayers() {
		if lineRegex.Instruction == layer.Instruction && lineRegex.Value.MatchString(layer.GetValue()) {
			dockerFileLine := fmt.Sprintf("%v %v", layer.GetInstruction(), layer.GetValue())
			violation := &v1.Alert_Violation{
				Message: fmt.Sprintf("Dockerfile Line '%v' matches the instruction '%v' and regex '%v'", dockerFileLine, layer.GetInstruction(), lineRegex.Value),
			}
			violations = append(violations, violation)
		}
	}
	return
}

func getVulnLink(v *v1.Vulnerability) string {
	if v.GetLink() != "" {
		return v.GetLink()
	}
	return cveLinkPrefix + v.GetCve()
}

func (policy *compiledImagePolicy) matchCVE(image *v1.Image) (violations []*v1.Alert_Violation, policyExists bool) {
	if policy.CVE == nil {
		return
	}
	policyExists = true
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			if policy.CVE.MatchString(vuln.GetCve()) {
				violations = append(violations, &v1.Alert_Violation{
					Message: fmt.Sprintf("CVE '%v' matches the regex '%+v'. Link: %v", vuln.GetCve(), policy.CVE, getVulnLink(vuln)),
				})
			}
		}
	}
	return
}

func (policy *compiledImagePolicy) matchCVSS(image *v1.Image) (violations []*v1.Alert_Violation, policyExists bool) {
	if policy.CVSS == nil {
		return
	}
	policyExists = true
	minimum := float32(math.MaxFloat32)
	var maximum float32
	var average float32

	var numVulns float32
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			minimum = min(minimum, vuln.GetCvss())
			maximum = max(maximum, vuln.GetCvss())
			average += vuln.GetCvss()
			numVulns++
		}
	}

	var value float32
	switch policy.CVSS.GetMathOp() {
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
	var comparatorChar string
	switch policy.CVSS.GetOp() {
	case v1.Comparator_LESS_THAN:
		comparatorFunc = func(x, y float32) bool { return x < y }
		comparatorChar = "<"
	case v1.Comparator_LESS_THAN_OR_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x <= y }
		comparatorChar = "<="
	case v1.Comparator_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x == y }
		comparatorChar = "="
	case v1.Comparator_GREATER_THAN_OR_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x >= y }
		comparatorChar = ">="
	case v1.Comparator_GREATER_THAN:
		comparatorFunc = func(x, y float32) bool { return x > y }
		comparatorChar = ">"
	}
	if comparatorFunc(value, policy.CVSS.GetValue()) {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("The %s(cvss) = %v. %v is %v threshold of %v", policy.CVSS.GetMathOp(), value, value, comparatorChar, policy.CVSS.GetValue()),
		})
	}
	return
}

func (policy *compiledImagePolicy) matchImageName(image *v1.Image) (violations []*v1.Alert_Violation, policyExists bool) {
	if policy.ImageNamePolicy == nil {
		return
	}
	policyExists = true
	if policy.ImageNamePolicy.Registry != nil && !policy.ImageNamePolicy.Registry.MatchString(image.Registry) {
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
	if policy.ImageNamePolicy.Namespace != nil && !policy.ImageNamePolicy.Namespace.MatchString(namespace) {
		return
	}
	if policy.ImageNamePolicy.Repo != nil && !policy.ImageNamePolicy.Repo.MatchString(repo) {
		return // return nothing if one of the regexes doesn't match. It must match all things in the image policy
	}
	if policy.ImageNamePolicy.Tag != nil && !policy.ImageNamePolicy.Tag.MatchString(image.Tag) {
		return
	}
	violations = append(violations, &v1.Alert_Violation{
		Message: fmt.Sprintf("Image name '%s' matches the name policy '%s'", images.Wrapper{Image: image}, policy.ImageNamePolicy),
	})
	return
}

func (policy *compiledImagePolicy) matchImageAge(image *v1.Image) (violations []*v1.Alert_Violation, policyExists bool) {
	if policy.ImageAgeDays == nil {
		return
	}
	policyExists = true
	deadline := time.Now().AddDate(0, 0, -int(*policy.ImageAgeDays))
	created := image.GetMetadata().GetCreated()
	if created == nil {
		return
	}
	createdTime, err := ptypes.Timestamp(created)
	if err != nil {
		log.Error(err) // Log just in case, though in reality this should not occur
	}
	if createdTime.Before(deadline) {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("Image Age '%v' is %0.2f days past the deadline", createdTime, deadline.Sub(createdTime).Hours()/24),
		})
	}
	return
}

func (policy *compiledImagePolicy) matchScanAge(image *v1.Image) (violations []*v1.Alert_Violation, policyExists bool) {
	if policy.ScanAgeDays == nil {
		return
	}
	policyExists = true
	deadline := time.Now().AddDate(0, 0, -int(*policy.ScanAgeDays))
	scanned := image.GetScan().GetScanTime()
	if scanned == nil {
		return
	}
	scannedTime, err := ptypes.Timestamp(scanned)
	if err != nil {
		log.Error(err) // Log just in case, though in reality this should not occur
	}
	if scannedTime.Before(deadline) {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("Scan Age '%v' is %0.2f days past the deadline", scannedTime, deadline.Sub(scannedTime).Hours()/24),
		})
	}
	return
}

func (policy *compiledImagePolicy) matchScanExists(image *v1.Image) (violations []*v1.Alert_Violation, policyExists bool) {
	if policy.ScanExists == nil {
		return
	}
	policyExists = true
	if *policy.ScanExists && image.GetScan() == nil {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("Image '%v' has not been scanned", images.Wrapper{Image: image}),
		})
	}
	return
}
