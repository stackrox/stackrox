package imageprocessor

import (
	"fmt"
	"regexp"
	"strings"

	"bitbucket.org/stack-rox/apollo/central/detection/processors"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
)

var (
	log = logging.LoggerForModule()
)

// compiledImagePolicy is an Image Policy that has been precompiled for matching deployments.
type compiledImagePolicy struct {
	Original *v1.Policy

	ImageNamePolicy *imageNamePolicyRegex

	ImageAgeDays *int64
	LineRule     *lineRuleFieldRegex

	CVSS        *v1.NumericalPolicy
	CVE         *regexp.Regexp
	Component   *componentRegex
	ScanAgeDays *int64
	ScanExists  *bool
}

type componentRegex struct {
	Name    *regexp.Regexp
	Version *regexp.Regexp
}

type lineRuleFieldRegex struct {
	Instruction string
	Value       *regexp.Regexp
}

type imageNamePolicyRegex struct {
	Registry  *regexp.Regexp
	Namespace *regexp.Regexp
	Repo      *regexp.Regexp
	Tag       *regexp.Regexp
}

func init() {
	processors.PolicySegmentCompilers = append(processors.PolicySegmentCompilers, NewCompiledImagePolicy)
}

// NewCompiledImagePolicy returns a new compiledImagePolicy.
func NewCompiledImagePolicy(policy *v1.Policy) (compiledP processors.CompiledPolicy, exist bool, err error) {
	imagePolicy := policy.GetImagePolicy()
	if imagePolicy == nil {
		return
	}
	exist = true

	var imageAge, scanAge *int64
	if imagePolicy.GetSetImageAgeDays() != nil {
		tmp := imagePolicy.GetImageAgeDays()
		imageAge = &tmp
	}
	if imagePolicy.GetSetScanAgeDays() != nil {
		tmp := imagePolicy.GetScanAgeDays()
		scanAge = &tmp
	}
	var scanExists *bool
	if imagePolicy.GetSetScanExists() != nil {
		tmp := imagePolicy.GetScanExists()
		scanExists = &tmp
	}

	if imagePolicy.GetCvss().GetValue() < 0 || imagePolicy.GetCvss().GetValue() > 10 {
		return nil, exist, fmt.Errorf("policy %s must have CVSS score between 0-10 (actual: %v)", policy.GetName(), imagePolicy.GetCvss().GetValue())
	}

	compiled := &compiledImagePolicy{
		Original:     policy,
		ImageAgeDays: imageAge,
		CVSS:         imagePolicy.GetCvss(),
		ScanAgeDays:  scanAge,
		ScanExists:   scanExists,
	}

	compiled.ImageNamePolicy, err = compileImageNamePolicyRegex(imagePolicy.GetImageName())
	if err != nil {
		return nil, exist, fmt.Errorf("image name: %s", err)
	}
	compiled.LineRule, err = compileLineRuleFieldRegex(imagePolicy.GetLineRule())
	if err != nil {
		return nil, exist, fmt.Errorf("image line: %s", err)
	}
	compiled.Component, err = compileComponent(imagePolicy.GetComponent())
	if err != nil {
		return nil, exist, fmt.Errorf("image component: %s", err)
	}
	compiled.CVE, err = processors.CompileStringRegex(imagePolicy.GetCve())
	if err != nil {
		return nil, exist, fmt.Errorf("image cve: %s", err)
	}

	return compiled, exist, nil
}

func compileImageNamePolicyRegex(policy *v1.ImageNamePolicy) (*imageNamePolicyRegex, error) {
	if policy == nil {
		return nil, nil
	}
	registry, err := processors.CompileStringRegex(policy.GetRegistry())
	if err != nil {
		return nil, err
	}
	namespace, err := processors.CompileStringRegex(policy.GetNamespace())
	if err != nil {
		return nil, err
	}
	repo, err := processors.CompileStringRegex(policy.GetRepo())
	if err != nil {
		return nil, err
	}
	tag, err := processors.CompileStringRegex(policy.GetTag())
	if err != nil {
		return nil, err
	}
	return &imageNamePolicyRegex{
		Registry:  registry,
		Namespace: namespace,
		Repo:      repo,
		Tag:       tag,
	}, nil
}

func compileComponent(comp *v1.ImagePolicy_Component) (*componentRegex, error) {
	if comp == nil {
		return nil, nil
	}
	name, err := processors.CompileStringRegex(comp.GetName())
	if err != nil {
		return nil, fmt.Errorf("component name '%v' is not a valid regex", comp.GetName())
	}
	version, err := processors.CompileStringRegex(comp.GetVersion())
	if err != nil {
		return nil, fmt.Errorf("component version '%v' is not a valid regex", comp.GetVersion())
	}
	return &componentRegex{
		Name:    name,
		Version: version,
	}, nil
}

func compileLineRuleFieldRegex(line *v1.DockerfileLineRuleField) (*lineRuleFieldRegex, error) {
	if line == nil {
		return nil, nil
	}
	if _, ok := registries.DockerfileInstructionSet[line.Instruction]; !ok {
		return nil, fmt.Errorf("%v is not a valid dockerfile instruction", line.Instruction)
	}
	value, err := processors.CompileStringRegex(line.Value)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("value must be defined for a dockerfile instruction")
	}
	return &lineRuleFieldRegex{
		Instruction: line.Instruction,
		Value:       value,
	}, nil
}

func (i imageNamePolicyRegex) String() string {
	var fields []string
	if i.Registry != nil {
		fields = append(fields, fmt.Sprintf("registry=%v", i.Registry))
	}
	if i.Namespace != nil {
		fields = append(fields, fmt.Sprintf("namespace=%v", i.Namespace))
	}
	if i.Repo != nil {
		fields = append(fields, fmt.Sprintf("repo=%v", i.Repo))
	}
	if i.Tag != nil {
		fields = append(fields, fmt.Sprintf("tag=%v", i.Tag))
	}
	return strings.Join(fields, ", ")
}
