package main

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	centralDebug "github.com/stackrox/rox/sensor/debugger/central"
)

type Condition interface {
	Check(msg *central.MsgFromSensor)
	Met() bool
	String() string
}

type CompletionChecker struct {
	conditions []Condition
	Done       concurrency.Signal
}

func NewCompletionChecker(cfg *Config, fakeCentral *centralDebug.FakeService) (*CompletionChecker, error) {
	cc := &CompletionChecker{
		Done: concurrency.NewSignal(),
	}

	for _, condCfg := range cfg.Completion.Conditions {
		cond, err := buildCondition(condCfg)
		if err != nil {
			return nil, err
		}
		cc.conditions = append(cc.conditions, cond)
	}

	fakeCentral.OnMessage(func(msg *central.MsgFromSensor) {
		for _, c := range cc.conditions {
			c.Check(msg)
		}
		if cc.allMet() {
			cc.Done.Signal()
		}
	})

	return cc, nil
}

func (cc *CompletionChecker) allMet() bool {
	for _, c := range cc.conditions {
		if !c.Met() {
			return false
		}
	}
	return true
}

func buildCondition(cfg ConditionConfig) (Condition, error) {
	switch cfg.Type {
	case "deploymentCount":
		return &deploymentCountCondition{target: cfg.Count}, nil
	case "deploymentField":
		return newDeploymentFieldCondition(cfg)
	default:
		return nil, fmt.Errorf("unknown condition type: %s", cfg.Type)
	}
}

type deploymentCountCondition struct {
	target int
	seen   int
}

func (c *deploymentCountCondition) Check(msg *central.MsgFromSensor) {
	if msg.GetEvent().GetDeployment() != nil {
		c.seen++
	}
}

func (c *deploymentCountCondition) Met() bool {
	return c.seen >= c.target
}

func (c *deploymentCountCondition) String() string {
	return fmt.Sprintf("deploymentCount: %d/%d", c.seen, c.target)
}

type deploymentFieldCondition struct {
	field   string
	value   string
	target  int
	matched int
}

func newDeploymentFieldCondition(cfg ConditionConfig) (*deploymentFieldCondition, error) {
	switch cfg.Field {
	case "permissionLevel", "exposure":
	default:
		return nil, fmt.Errorf("unsupported deployment field: %s", cfg.Field)
	}
	return &deploymentFieldCondition{
		field:  cfg.Field,
		value:  cfg.Value,
		target: cfg.Count,
	}, nil
}

func (c *deploymentFieldCondition) Check(msg *central.MsgFromSensor) {
	dep := msg.GetEvent().GetDeployment()
	if dep == nil {
		return
	}
	if c.matches(dep) {
		c.matched++
	}
}

func (c *deploymentFieldCondition) matches(dep *storage.Deployment) bool {
	switch c.field {
	case "permissionLevel":
		return strings.EqualFold(dep.GetServiceAccountPermissionLevel().String(), c.value)
	case "exposure":
		for _, port := range dep.GetPorts() {
			if strings.EqualFold(port.GetExposure().String(), c.value) {
				return true
			}
		}
		return false
	}
	return false
}

func (c *deploymentFieldCondition) Met() bool {
	return c.matched >= c.target
}

func (c *deploymentFieldCondition) String() string {
	return fmt.Sprintf("deploymentField(%s=%s): %d/%d", c.field, c.value, c.matched, c.target)
}
