package configurationprocessor

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"bitbucket.org/stack-rox/apollo/central/detection/processors"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

// compiledConfigurationPolicy is a Configuration Policy that has been precompiled for matching deployments.
type compiledConfigurationPolicy struct {
	Original *v1.Policy

	Env                *compiledEnvironmentPolicy
	RequiredLabel      *requiredLabelPolicy
	RequiredAnnotation *requiredAnnotationPolicy

	Args      *regexp.Regexp
	Command   *regexp.Regexp
	Directory *regexp.Regexp
	User      *regexp.Regexp

	Volume *compiledVolumePolicy
	Port   *compiledPortPolicy
}

type keyValuePolicy struct {
	Key   *regexp.Regexp
	Value *regexp.Regexp
}

type requiredLabelPolicy struct {
	*keyValuePolicy
}

type requiredAnnotationPolicy struct {
	*keyValuePolicy
}

type compiledEnvironmentPolicy struct {
	*keyValuePolicy
}

type compiledVolumePolicy struct {
	Name        *regexp.Regexp
	Source      *regexp.Regexp
	Destination *regexp.Regexp
	ReadOnly    *bool
	Type        *regexp.Regexp
}

type compiledPortPolicy struct {
	Port     int32
	Protocol string
}

func init() {
	processors.PolicySegmentCompilers = append(processors.PolicySegmentCompilers, NewCompiledConfigurationPolicy)
}

// NewCompiledConfigurationPolicy returns a new compiledConfigurationPolicy.
func NewCompiledConfigurationPolicy(policy *v1.Policy) (compiledP processors.CompiledPolicy, err error) {
	configurationPolicy := policy.GetFields()
	compiled := new(compiledConfigurationPolicy)
	compiled.Original = policy

	compiled.Env, err = newCompiledEnvironmentPolicy(configurationPolicy.GetEnv())
	if err != nil {
		return nil, fmt.Errorf("env: %s", err)
	}

	compiled.RequiredLabel, err = newRequiredLabelPolicy(configurationPolicy.GetRequiredLabel())
	if err != nil {
		return nil, fmt.Errorf("missing label: %s", err)
	}

	compiled.RequiredAnnotation, err = newRequiredAnnotationPolicy(configurationPolicy.GetRequiredAnnotation())
	if err != nil {
		return nil, fmt.Errorf("missing annotation: %s", err)
	}

	compiled.Args, err = processors.CompileStringRegex(configurationPolicy.GetArgs())
	if err != nil {
		return nil, fmt.Errorf("args: %s", err)
	}

	compiled.Command, err = processors.CompileStringRegex(configurationPolicy.GetCommand())
	if err != nil {
		return nil, fmt.Errorf("command: %s", err)
	}

	compiled.Directory, err = processors.CompileStringRegex(configurationPolicy.GetDirectory())
	if err != nil {
		return nil, fmt.Errorf("directory: %s", err)
	}

	compiled.User, err = processors.CompileStringRegex(configurationPolicy.GetUser())
	if err != nil {
		return nil, fmt.Errorf("user: %s", err)
	}

	compiled.Volume, err = newCompiledVolumePolicy(configurationPolicy.GetVolumePolicy())
	if err != nil {
		return nil, fmt.Errorf("volume: %s", err)
	}

	compiled.Port = newCompiledPortPolicy(configurationPolicy.GetPortPolicy())
	return compiled, nil
}

func newCompiledEnvironmentPolicy(kvPolicy *v1.KeyValuePolicy) (compiled *compiledEnvironmentPolicy, err error) {
	if kvPolicy == nil {
		return
	}
	requiredKVPolicy, err := newRequiredKeyValuePolicy(kvPolicy)
	if err != nil {
		return nil, err
	}
	return &compiledEnvironmentPolicy{
		keyValuePolicy: requiredKVPolicy,
	}, nil
}

func newRequiredLabelPolicy(kvPolicy *v1.KeyValuePolicy) (compiled *requiredLabelPolicy, err error) {
	if kvPolicy == nil {
		return
	}
	requiredKVPolicy, err := newRequiredKeyValuePolicy(kvPolicy)
	if err != nil {
		return nil, err
	}
	return &requiredLabelPolicy{
		keyValuePolicy: requiredKVPolicy,
	}, nil
}

func newRequiredAnnotationPolicy(kvPolicy *v1.KeyValuePolicy) (compiled *requiredAnnotationPolicy, err error) {
	if kvPolicy == nil {
		return
	}
	requiredKVPolicy, err := newRequiredKeyValuePolicy(kvPolicy)
	if err != nil {
		return nil, err
	}
	return &requiredAnnotationPolicy{
		keyValuePolicy: requiredKVPolicy,
	}, nil
}

func newRequiredKeyValuePolicy(kvPolicy *v1.KeyValuePolicy) (compiled *keyValuePolicy, err error) {
	if kvPolicy == nil {
		return
	}
	if kvPolicy.GetKey() == "" && kvPolicy.GetValue() == "" {
		return nil, errors.New("Both key and value cannot be empty")
	}

	compiled = new(keyValuePolicy)
	compiled.Key, err = processors.CompileStringRegex(kvPolicy.GetKey())
	if err != nil {
		return
	}

	compiled.Value, err = processors.CompileStringRegex(kvPolicy.GetValue())
	return
}

func newCompiledVolumePolicy(volumePolicy *v1.VolumePolicy) (compiled *compiledVolumePolicy, err error) {
	if volumePolicy == nil || (!volumePolicy.GetReadOnly() && volumePolicy.GetName() == "" && volumePolicy.GetSource() == "" && volumePolicy.GetDestination() == "" && volumePolicy.GetType() == "") {
		return
	}

	compiled = new(compiledVolumePolicy)
	if volumePolicy.GetSetReadOnly() != nil {
		readOnly := volumePolicy.GetReadOnly()
		compiled.ReadOnly = &readOnly
	}

	compiled.Name, err = processors.CompileStringRegex(volumePolicy.GetName())
	if err != nil {
		return nil, fmt.Errorf("name: %s", err)
	}

	compiled.Source, err = processors.CompileStringRegex(volumePolicy.GetSource())
	if err != nil {
		return nil, fmt.Errorf("source: %s", err)
	}

	compiled.Destination, err = processors.CompileStringRegex(volumePolicy.GetDestination())
	if err != nil {
		return nil, fmt.Errorf("destination: %s", err)
	}

	compiled.Type, err = processors.CompileStringRegex(volumePolicy.GetType())
	if err != nil {
		return nil, fmt.Errorf("type: %s", err)
	}

	return
}

func newCompiledPortPolicy(portPolicy *v1.PortPolicy) *compiledPortPolicy {
	if portPolicy == nil {
		return nil
	}

	return &compiledPortPolicy{
		Port:     portPolicy.GetPort(),
		Protocol: portPolicy.GetProtocol(),
	}
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
	if p.Source != nil {
		fields = append(fields, fmt.Sprintf("source=%v", p.Source))
	}
	if p.Destination != nil {
		fields = append(fields, fmt.Sprintf("destination=%v", p.Destination))
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
