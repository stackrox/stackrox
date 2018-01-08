package configurationprocessor

import (
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

type matchFunc func(*v1.Container) ([]*v1.Alert_Violation, bool)

func (p *compiledConfigurationPolicy) Match(deployment *v1.Deployment, container *v1.Container) (output []*v1.Alert_Violation) {
	matchFunctions := []matchFunc{
		p.matchConfigs,
		p.Env.match,
		p.Volume.match,
		p.Port.match,
	}

	var violations, vs []*v1.Alert_Violation
	var exists bool

	// Every sub-policy that exists must match and return violations for the policy to match.
	for _, f := range matchFunctions {
		if vs, exists = f(container); exists && len(vs) == 0 {
			return
		}
		violations = append(violations, vs...)
	}

	output = violations
	return
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
	if p.Path != nil && !p.Path.MatchString(vol.GetPath()) {
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
