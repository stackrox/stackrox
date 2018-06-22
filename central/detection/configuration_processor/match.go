package configurationprocessor

import (
	"fmt"
	"math"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

type containerMatchFunc func(*v1.Container) ([]*v1.Alert_Violation, bool)
type deploymentMatchFunc func(*v1.Deployment) ([]*v1.Alert_Violation, bool)

func (p *compiledConfigurationPolicy) MatchDeployment(deployment *v1.Deployment) ([]*v1.Alert_Violation, bool) {
	matchFunctions := []deploymentMatchFunc{
		p.RequiredLabel.match,
		p.RequiredAnnotation.match,
		p.matchTotalResourcePolicy,
	}

	var violations []*v1.Alert_Violation
	var exists bool
	// Every sub-policy that exists must match and return violations for the policy to match.
	for _, f := range matchFunctions {
		vs, valid := f(deployment)
		if valid && len(vs) == 0 {
			return nil, true
		} else if valid {
			exists = true
		}
		violations = append(violations, vs...)
	}

	return violations, exists
}

func (p *compiledConfigurationPolicy) MatchContainer(container *v1.Container) ([]*v1.Alert_Violation, bool) {
	matchFunctions := []containerMatchFunc{
		p.matchConfigs,
		p.Env.match,
		p.Volume.match,
		p.Port.match,
		p.matchTopLevelContainerResourcePolicy,
	}

	var violations []*v1.Alert_Violation
	var exists bool

	// Every sub-policy that exists must match and return violations for the policy to match.
	for _, f := range matchFunctions {
		vs, valid := f(container)
		if valid && len(vs) == 0 {
			return nil, true
		} else if valid {
			exists = true
		}
		violations = append(violations, vs...)
	}
	return violations, exists
}

func (p *compiledConfigurationPolicy) matchConfigs(container *v1.Container) (violations []*v1.Alert_Violation, exists bool) {
	if p.Args == nil && p.Command == nil && p.Directory == nil && p.User == nil {
		return
	}

	exists = true
	config := container.GetConfig()

	if p.Args != nil && !p.matchArg(config.GetArgs()) {
		return
	}
	if p.Command != nil && !p.matchCommand(config.GetCommand()) {
		return
	}
	if p.Directory != nil && !p.Directory.MatchString(config.GetDirectory()) {
		return
	}
	if p.User != nil && !p.User.MatchString(config.GetUser()) {
		return
	}

	violations = append(violations, &v1.Alert_Violation{
		Message: fmt.Sprintf("Container Configuration %s matched configured policy %s", configWrap{config}, p),
	})

	return
}

func (p *compiledConfigurationPolicy) matchArg(args []string) bool {
	for _, arg := range args {
		if p.Args.MatchString(arg) {
			return true
		}
	}

	return false
}

func (p *compiledConfigurationPolicy) matchCommand(commands []string) bool {
	for _, command := range commands {
		if p.Command.MatchString(command) {
			return true
		}
	}

	return false
}

func (p *compiledEnvironmentPolicy) match(container *v1.Container) (violations []*v1.Alert_Violation, exists bool) {
	if p == nil {
		return
	}

	config := container.GetConfig()
	exists = true

	for _, env := range config.GetEnv() {
		if p.Key != nil && p.Value != nil {
			if p.Key.MatchString(env.GetKey()) && p.Value.MatchString(env.GetValue()) {
				violations = append(violations, &v1.Alert_Violation{
					Message: fmt.Sprintf("Container Environment (key='%s', value='%s') matched configured policy (key='%s', value='%s')", env.GetKey(), env.GetValue(), p.Key, p.Value),
				})
			}
		} else if p.Key != nil {
			if p.Key.MatchString(env.GetKey()) {
				violations = append(violations, &v1.Alert_Violation{
					Message: fmt.Sprintf("Container Environment (key='%s', value='%s') matched configured policy (key='%s')", env.GetKey(), env.GetValue(), p.Key),
				})
			}
		} else if p.Value != nil {
			if p.Value.MatchString(env.GetValue()) {
				violations = append(violations, &v1.Alert_Violation{
					Message: fmt.Sprintf("Container Environment (key='%s', value='%s') matched configured policy (value='%s')", env.GetKey(), env.GetValue(), p.Value),
				})
			}
		}
	}

	return
}

func (p *requiredAnnotationPolicy) match(deployment *v1.Deployment) (violations []*v1.Alert_Violation, exists bool) {
	if p == nil {
		return
	}
	exists = true
	violations = matchRequiredKeyValue(deployment.GetAnnotations(), p.keyValuePolicy, "annotation")
	return
}

func (p *requiredLabelPolicy) match(deployment *v1.Deployment) (violations []*v1.Alert_Violation, exists bool) {
	if p == nil {
		return
	}
	exists = true
	violations = matchRequiredKeyValue(deployment.GetLabels(), p.keyValuePolicy, "label")
	return
}

func matchRequiredKeyValue(deploymentKeyValues []*v1.Deployment_KeyValue, policy *keyValuePolicy, name string) []*v1.Alert_Violation {
	for _, keyValue := range deploymentKeyValues {
		if policy.Key != nil && policy.Value != nil {
			if policy.Key.MatchString(keyValue.GetKey()) && policy.Value.MatchString(keyValue.GetValue()) {
				return nil
			}
		} else if policy.Key != nil {
			if policy.Key.MatchString(keyValue.GetKey()) {
				return nil
			}
		} else if policy.Value != nil {
			if policy.Value.MatchString(keyValue.GetValue()) {
				return nil
			}
		}
	}
	var fields []string
	if policy.Key != nil {
		fields = append(fields, fmt.Sprintf("key='%s'", policy.Key))
	}
	if policy.Value != nil {
		fields = append(fields, fmt.Sprintf("value='%s'", policy.Value))
	}
	return []*v1.Alert_Violation{
		{
			Message: fmt.Sprintf("Could not find %s that matched required %s policy (%s)", name, name, strings.Join(fields, ",")),
		},
	}
}

func (p *compiledVolumePolicy) match(container *v1.Container) (violations []*v1.Alert_Violation, exists bool) {
	if p == nil {
		return
	}

	volumes := container.GetVolumes()
	exists = true

	for _, vol := range volumes {
		violations = append(violations, p.matchVolume(vol)...)
	}

	return
}

func (p *compiledVolumePolicy) matchVolume(vol *v1.Volume) (violations []*v1.Alert_Violation) {
	if p.ReadOnly != nil && vol.GetReadOnly() != *p.ReadOnly {
		return
	}
	if p.Name != nil && !p.Name.MatchString(vol.GetName()) {
		return
	}
	if p.Source != nil && !p.Source.MatchString(vol.GetSource()) {
		return
	}
	if p.Destination != nil && !p.Destination.MatchString(vol.GetDestination()) {
		return
	}
	if p.Type != nil && !p.Type.MatchString(vol.GetType()) {
		return
	}

	violations = append(violations, &v1.Alert_Violation{
		Message: fmt.Sprintf("Volume %+v matched configured policy %s", vol, p),
	})

	return
}

func (p *compiledPortPolicy) match(container *v1.Container) (violations []*v1.Alert_Violation, exists bool) {
	if p == nil {
		return
	}

	ports := container.GetPorts()
	exists = true

	for _, port := range ports {
		violations = append(violations, p.matchPort(port)...)
	}

	return
}

func (p *compiledPortPolicy) matchPort(port *v1.PortConfig) (violations []*v1.Alert_Violation) {
	if p.Port != 0 && p.Port != port.GetContainerPort() {
		return
	}

	if p.Protocol != "" && !strings.EqualFold(p.Protocol, port.GetProtocol()) {
		return
	}

	violations = append(violations, &v1.Alert_Violation{
		Message: fmt.Sprintf("Port %+v matched configured policy %s", port, p),
	})

	return
}

func matchNumericalPolicy(prefix, id string, value float32, p *v1.ResourcePolicy_NumericalPolicy) (violations []*v1.Alert_Violation, policyExists bool) {
	if p == nil {
		return
	}
	policyExists = true
	var comparatorFunc func(x, y float32) bool
	var comparatorString string
	switch p.GetOp() {
	case v1.Comparator_LESS_THAN:
		comparatorFunc = func(x, y float32) bool { return x < y }
		comparatorString = "less than"
	case v1.Comparator_LESS_THAN_OR_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x <= y }
		comparatorString = "less than or equal to"
	case v1.Comparator_EQUALS:
		comparatorFunc = func(x, y float32) bool { return math.Abs(float64(x-y)) <= 1e-5 }
		comparatorString = "equal to"
	case v1.Comparator_GREATER_THAN_OR_EQUALS:
		comparatorFunc = func(x, y float32) bool { return x >= y }
		comparatorString = "greater than or equal to"
	case v1.Comparator_GREATER_THAN:
		comparatorFunc = func(x, y float32) bool { return x > y }
		comparatorString = "greater than"
	}
	if comparatorFunc(value, p.GetValue()) {
		violations = append(violations, &v1.Alert_Violation{
			Message: fmt.Sprintf("The %s of %0.2f for %s is %s the threshold of %v", prefix, value,
				id, comparatorString, p.GetValue()),
		})
	}
	return
}

func (p *compiledConfigurationPolicy) matchTotalResourcePolicy(deployment *v1.Deployment) (violations []*v1.Alert_Violation, policyExists bool) {
	var resource v1.Resources
	for _, c := range deployment.GetContainers() {
		resource.CpuCoresRequest += c.GetResources().GetCpuCoresRequest() * float32(deployment.GetReplicas())
		resource.CpuCoresLimit += c.GetResources().GetCpuCoresLimit() * float32(deployment.GetReplicas())
		resource.MemoryMbRequest += c.GetResources().GetMemoryMbRequest() * float32(deployment.GetReplicas())
		resource.MemoryMbLimit += c.GetResources().GetMemoryMbLimit() * float32(deployment.GetReplicas())
	}

	return p.matchResources(p.TotalResources, &resource, "deployment")
}

func (p *compiledConfigurationPolicy) matchTopLevelContainerResourcePolicy(container *v1.Container) (violations []*v1.Alert_Violation, policyExists bool) {
	return p.matchResources(p.ContainerResources, container.GetResources(), fmt.Sprintf("container %s", container.GetImage().GetName().GetRemote()))
}

func (p *compiledConfigurationPolicy) matchResources(policy *v1.ResourcePolicy, resource *v1.Resources, identifier string) (violations []*v1.Alert_Violation, policyExists bool) {
	if policy == nil {
		return
	}
	policyExists = true
	matchFunctions := []func(*v1.ResourcePolicy, *v1.Resources, string) ([]*v1.Alert_Violation, bool){
		p.matchCPUResourceRequest,
		p.matchCPUResourceLimit,
		p.matchMemoryResourceRequest,
		p.matchMemoryResourceLimit,
	}

	// OR the violations together
	for _, f := range matchFunctions {
		vs, _ := f(policy, resource, identifier)
		violations = append(violations, vs...)
	}
	return
}

func (p *compiledConfigurationPolicy) matchCPUResourceRequest(rp *v1.ResourcePolicy, resources *v1.Resources, id string) (violations []*v1.Alert_Violation, policyExists bool) {
	violations, policyExists = matchNumericalPolicy("CPU resource request",
		id, resources.GetCpuCoresRequest(), rp.GetCpuResourceRequest())
	return
}

func (p *compiledConfigurationPolicy) matchCPUResourceLimit(rp *v1.ResourcePolicy, resources *v1.Resources, id string) (violations []*v1.Alert_Violation, policyExists bool) {
	violations, policyExists = matchNumericalPolicy("CPU resource limit",
		id, resources.GetCpuCoresLimit(), rp.GetCpuResourceLimit())
	return
}

func (p *compiledConfigurationPolicy) matchMemoryResourceRequest(rp *v1.ResourcePolicy, resources *v1.Resources, id string) (violations []*v1.Alert_Violation, policyExists bool) {
	violations, policyExists = matchNumericalPolicy("Memory resource request",
		id, resources.GetMemoryMbRequest(), rp.GetMemoryResourceRequest())
	return
}

func (p *compiledConfigurationPolicy) matchMemoryResourceLimit(rp *v1.ResourcePolicy, resources *v1.Resources, id string) (violations []*v1.Alert_Violation, policyExists bool) {
	violations, policyExists = matchNumericalPolicy("Memory resource limit",
		id, resources.GetMemoryMbLimit(), rp.GetMemoryResourceLimit())
	return
}
