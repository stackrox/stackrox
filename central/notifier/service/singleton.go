package service

import (
	"github.com/stackrox/rox/central/integrationhealth/reporter"
	"github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/policycleaner"
	"github.com/stackrox/rox/central/notifier/processor"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	cryptoKey := ""
	if env.EncNotifierCreds.BooleanSetting() {
		var err error
		cryptoKey, err = notifierUtils.GetNotifierSecretEncryptionKey()
		if err != nil {
			utils.CrashOnError(err)
		}
	}

	as = New(
		datastore.Singleton(),
		processor.Singleton(),
		policycleaner.Singleton(),
		reporter.Singleton(),
		cryptoutils.NewGCMCryptoCodec(),
		cryptoKey,
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
