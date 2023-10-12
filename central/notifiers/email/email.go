package email

import (
	"github.com/stackrox/rox/central/notifiers/metadatagetter"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	mitreDS "github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/notifiers/email"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	cryptoKey := ""
	var err error
	if env.EncNotifierCreds.BooleanSetting() {
		cryptoKey, err = notifierUtils.GetNotifierSecretEncryptionKey()
		if err != nil {
			utils.CrashOnError(err)
		}
	}
	notifiers.Add(notifiers.EmailType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		e, err := email.NewEmail(notifier, metadatagetter.Singleton(), mitreDS.Singleton(), cryptoKey)
		return e, err
	})
}
