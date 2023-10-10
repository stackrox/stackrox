package syslog

import (
	"github.com/stackrox/rox/central/notifiers/metadatagetter"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/notifiers/syslog"
)

func init() {
	notifiers.Add(notifiers.SyslogType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		return syslog.NewSyslog(notifier, metadatagetter.Singleton())
	})
}
