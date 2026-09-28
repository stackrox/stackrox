package cmd

import (
	"fmt"

	"github.com/coreos/go-systemd/v22/daemon"
)

// notifySystemdReady sends READY=1 on NOTIFY_SOCKET when set (Type=notify /
// Quadlet Notify=true). No-op when unset so bare `roxagent serve` still works.
func notifySystemdReady() error {
	sent, err := daemon.SdNotify(false, daemon.SdNotifyReady)
	if err != nil {
		return fmt.Errorf("sd_notify READY=1: %w", err)
	}
	if sent {
		log.Info("Signaled systemd readiness (READY=1)")
	}
	return nil
}
