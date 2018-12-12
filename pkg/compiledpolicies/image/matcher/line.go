package matcher

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compiledpolicies/utils"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
)

func init() {
	compilers = append(compilers, newLineMatcher)
}

func newLineMatcher(policy *storage.Policy) (Matcher, error) {
	line := policy.GetFields().GetLineRule()
	if line == nil {
		return nil, nil
	}
	if _, ok := registryTypes.DockerfileInstructionSet[line.Instruction]; !ok {
		return nil, fmt.Errorf("%v is not a valid dockerfile instruction", line.Instruction)
	}
	lineRegex, err := utils.CompileStringRegex(line.Value)
	if err != nil {
		return nil, err
	}
	if lineRegex == nil {
		return nil, fmt.Errorf("value must be defined for a dockerfile instruction")
	}
	matcher := &lineMatcherImpl{instruction: line.Instruction, lineRegex: lineRegex}
	return matcher.match, nil
}

type lineMatcherImpl struct {
	instruction string
	lineRegex   *regexp.Regexp
}

func (p *lineMatcherImpl) match(image *storage.Image) (violations []*v1.Alert_Violation) {
	for _, layer := range image.GetMetadata().GetV1().GetLayers() {
		if p.instruction == layer.Instruction && p.lineRegex.MatchString(layer.GetValue()) {
			dockerFileLine := fmt.Sprintf("%v %v", layer.GetInstruction(), layer.GetValue())
			violation := &v1.Alert_Violation{
				Message: fmt.Sprintf("Dockerfile Line '%v' matches the instruction '%v' and regex '%v'", dockerFileLine, layer.GetInstruction(), p.lineRegex),
			}
			violations = append(violations, violation)
		}
	}
	return
}
