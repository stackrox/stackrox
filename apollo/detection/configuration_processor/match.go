package configurationprocessor

import (
	"fmt"
	"regexp"
	"strings"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type compiledConfigurationPolicy struct {
	Original *v1.Policy

	Env *compiledEnvironmentPolicy

	Args      *regexp.Regexp
	Command   *regexp.Regexp
	Directory *regexp.Regexp
	User      *regexp.Regexp

	Volume *compiledVolumePolicy
	Port   *compiledPortPolicy
}

type compiledEnvironmentPolicy struct {
	Key   *regexp.Regexp
	Value *regexp.Regexp
}

type compiledVolumePolicy struct {
	Name     *regexp.Regexp
	Path     *regexp.Regexp
	ReadOnly *bool
	Type     *regexp.Regexp
}

type compiledPortPolicy struct {
	Port     int32
	Protocol string
}

func newCompiledConfigurationPolicy(policy *v1.Policy) (compiled *compiledConfigurationPolicy, err error) {
	if policy.GetConfigurationPolicy() == nil {
		return nil, fmt.Errorf("policy %s must contain container configuration policy", policy.GetName())
	}

	configurationPolicy := policy.GetConfigurationPolicy()
	compiled = new(compiledConfigurationPolicy)
	compiled.Original = policy

	compiled.Env, err = newCompiledEnvironmentPolicy(configurationPolicy.GetEnv())
	if err != nil {
		return nil, fmt.Errorf("env: %s", err)
	}

	compiled.Args, err = compileStringRegex(configurationPolicy.GetArgs())
	if err != nil {
		return nil, fmt.Errorf("args: %s", err)
	}

	compiled.Command, err = compileStringRegex(configurationPolicy.GetCommand())
	if err != nil {
		return nil, fmt.Errorf("command: %s", err)
	}

	compiled.Directory, err = compileStringRegex(configurationPolicy.GetDirectory())
	if err != nil {
		return nil, fmt.Errorf("directory: %s", err)
	}

	compiled.User, err = compileStringRegex(configurationPolicy.GetUser())
	if err != nil {
		return nil, fmt.Errorf("user: %s", err)
	}

	compiled.Volume, err = newCompiledVolumePolicy(configurationPolicy.GetVolumePolicy())
	if err != nil {
		return nil, fmt.Errorf("volume: %s", err)
	}

	compiled.Port = newCompiledPortPolicy(configurationPolicy.GetPortPolicy())
	return
}

func newCompiledEnvironmentPolicy(envPolicy *v1.ConfigurationPolicy_EnvironmentPolicy) (compiled *compiledEnvironmentPolicy, err error) {
	if envPolicy == nil || (envPolicy.GetKey() == "" && envPolicy.GetValue() == "") {
		return
	}

	compiled = new(compiledEnvironmentPolicy)
	compiled.Key, err = compileStringRegex(envPolicy.GetKey())
	if err != nil {
		return
	}

	compiled.Value, err = compileStringRegex(envPolicy.GetValue())
	return
}

func newCompiledVolumePolicy(volumePolicy *v1.ConfigurationPolicy_VolumePolicy) (compiled *compiledVolumePolicy, err error) {
	if volumePolicy == nil || (!volumePolicy.GetReadOnly() && volumePolicy.GetName() == "" && volumePolicy.GetPath() == "" && volumePolicy.GetType() == "") {
		return
	}

	compiled = new(compiledVolumePolicy)
	if volumePolicy.GetSetReadOnly() != nil {
		readOnly := volumePolicy.GetReadOnly()
		compiled.ReadOnly = &readOnly
	}

	compiled.Name, err = compileStringRegex(volumePolicy.GetName())
	if err != nil {
		return nil, fmt.Errorf("name: %s", err)
	}

	compiled.Path, err = compileStringRegex(volumePolicy.GetPath())
	if err != nil {
		return nil, fmt.Errorf("path: %s", err)
	}

	compiled.Type, err = compileStringRegex(volumePolicy.GetType())
	if err != nil {
		return nil, fmt.Errorf("type: %s", err)
	}

	return
}

func newCompiledPortPolicy(portPolicy *v1.ConfigurationPolicy_PortPolicy) *compiledPortPolicy {
	if portPolicy == nil {
		return nil
	}

	return &compiledPortPolicy{
		Port:     portPolicy.GetPort(),
		Protocol: portPolicy.GetProtocol(),
	}
}

func compileStringRegex(regex string) (*regexp.Regexp, error) {
	if regex == "" {
		return nil, nil
	}
	return regexp.Compile(regex)
}

func (p *compiledConfigurationPolicy) String() string {
	var fields []string
	if p.Args != nil {
		fields = append(fields, fmt.Sprintf("args=%v", p.Args))
	}
	if p.Command != nil {
		fields = append(fields, fmt.Sprintf("command=%v", p.Command))
	}
	if p.Directory != nil {
		fields = append(fields, fmt.Sprintf("directory=%v", p.Directory))
	}
	if p.User != nil {
		fields = append(fields, fmt.Sprintf("user=%v", p.User))
	}
	return strings.Join(fields, ", ")
}

func (p *compiledVolumePolicy) String() string {
	var fields []string
	if p.ReadOnly != nil {
		fields = append(fields, fmt.Sprintf("read_only=%t", *p.ReadOnly))
	}
	if p.Name != nil {
		fields = append(fields, fmt.Sprintf("name=%v", p.Name))
	}
	if p.Path != nil {
		fields = append(fields, fmt.Sprintf("path=%v", p.Path))
	}
	if p.Type != nil {
		fields = append(fields, fmt.Sprintf("type=%v", p.Type))
	}
	return strings.Join(fields, ", ")
}

func (p *compiledPortPolicy) String() string {
	var fields []string
	if p.Port != 0 {
		fields = append(fields, fmt.Sprintf("port=%v", p.Port))
	}
	if p.Protocol != "" {
		fields = append(fields, fmt.Sprintf("protocol=%v", p.Protocol))
	}
	return strings.Join(fields, ", ")
}

type configWrap struct {
	*v1.ContainerConfig
}

func (c configWrap) String() string {
	var fields []string
	if len(c.GetArgs()) != 0 {
		fields = append(fields, fmt.Sprintf("args=%v", c.Args))
	}
	if len(c.Command) != 0 {
		fields = append(fields, fmt.Sprintf("command=%v", c.Command))
	}
	if c.Directory != "" {
		fields = append(fields, fmt.Sprintf("directory=%v", c.Directory))
	}
	if c.User != "" {
		fields = append(fields, fmt.Sprintf("user=%v", c.User))
	}
	return strings.Join(fields, ", ")
}

type matchFunc func(*v1.Container) ([]*v1.Alert_Violation, bool)

// Match checks whether a policy matches a given deployment.
// Each container is considered independently.
func (p *compiledConfigurationPolicy) match(deployment *v1.Deployment) (violations []*v1.Alert_Violation) {
	for _, c := range deployment.GetContainers() {
		violations = append(violations, p.matchContainer(c)...)
	}

	return
}

func (p *compiledConfigurationPolicy) matchContainer(container *v1.Container) (output []*v1.Alert_Violation) {
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
