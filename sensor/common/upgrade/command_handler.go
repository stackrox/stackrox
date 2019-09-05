package upgrade

import "github.com/stackrox/rox/generated/internalapi/central"

// CommandHandler handles commands relating to sensor upgrades.
type CommandHandler interface {
	Start()
	Stop()
	SendCommand(trigger *central.SensorUpgradeTrigger) bool
}
