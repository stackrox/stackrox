package acscsemail

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/utils"
)

func newACSCSEmail(notifier *storage.Notifier, cryptoCodec cryptocodec.CryptoCodec, cryptoKey string) (*acscsEmail, error) {
	return &acscsEmail{
		notifier: notifier,
	}, nil
}

type acscsEmail struct {
	notifier *storage.Notifier
}

func (*acscsEmail) Close(context.Context) error {
	return nil
}

func (e *acscsEmail) ProtoNotifier() *storage.Notifier {
	return e.notifier
}

// Test sends a test notification.
func (e *acscsEmail) Test(ctx context.Context) *notifiers.NotifierError {
	return notifiers.NewNotifierError("TODO: not implemented", nil)
}

func (e *acscsEmail) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	return errors.New("TODO: not implemented")
}

func (e *acscsEmail) NetworkPolicyYAMLNotify(ctx context.Context, yaml string, clusterName string) error {
	return errors.New("TODO: not implemented")
}

func (e *acscsEmail) ReportNotify(ctx context.Context, zippedReportData *bytes.Buffer, recipients []string, subject, messageText string) error {
	return errors.New("TODO: not implemented")
}

func init() {
	if !features.ACSCSEmailNotifier.Enabled() || !env.ManagedCentral.BooleanSetting() {
		return
	}

	cryptoKey := ""
	var err error
	if env.EncNotifierCreds.BooleanSetting() {
		cryptoKey, _, err = notifierUtils.GetActiveNotifierEncryptionKey()
		if err != nil {
			utils.Should(errors.Wrap(err, "Error reading encryption key, notifier will be unable to send notifications"))
		}
	}

	notifiers.Add(notifiers.ACSCSEmailType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		g, err := newACSCSEmail(notifier, cryptocodec.Singleton(), cryptoKey)
		return g, err
	})
}
