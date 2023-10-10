package email

import (
	"github.com/stackrox/rox/central/notifiers/metadatagetter"
	"github.com/stackrox/rox/generated/storage"
	mitreDS "github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/notifiers/email"
)

func init() {
	notifiers.Add(notifiers.EmailType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		e, err := email.NewEmail(notifier, metadatagetter.Singleton(), mitreDS.Singleton())
		return e, err
	})
}
