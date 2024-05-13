package acscsemail

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers/email"
	"github.com/stackrox/rox/central/notifiers/metadatagetter"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	mitreDS "github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/utils"
)

type AcscsMessage struct {
	To         []string `json:"to"`
	RawMessage []byte   `json:"rawMessage"`
}

func newACSCSEmail(notifier *storage.Notifier, client Client, metadataGetter notifiers.MetadataGetter, mitreStore mitreDS.AttackReadOnlyDataStore,
	cryptoCodec cryptocodec.CryptoCodec, cryptoKey string) (*acscsEmail, error) {
	return &acscsEmail{
		notifier:       notifier,
		client:         client,
		metadataGetter: metadataGetter,
		mitreStore:     mitreStore,
	}, nil
}

type acscsEmail struct {
	notifier       *storage.Notifier
	client         Client
	metadataGetter notifiers.MetadataGetter
	mitreStore     mitreDS.AttackReadOnlyDataStore
}

func (*acscsEmail) Close(context.Context) error {
	return nil
}

func (e *acscsEmail) ProtoNotifier() *storage.Notifier {
	return e.notifier
}

// Test sends a test notification.
func (e *acscsEmail) Test(ctx context.Context) *notifiers.NotifierError {
	subject := "RHACS Cloud Service Test Email"
	body := fmt.Sprintf("%v\r\n", "This is a test email created to test integration with ACSCS email service")
	msg := email.Message{
		To:      []string{e.notifier.GetLabelDefault()},
		Subject: subject,
		Body:    body,
	}

	if err := e.send(ctx, &msg); err != nil {
		return notifiers.NewNotifierError("failed to send test message to ACSCS email service", err)
	}

	return nil
}

// AlertNotify takes in an alert, generates a message from it and sends it to the ACSCS email service
func (e *acscsEmail) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	subject := notifiers.SummaryForAlert(alert)
	body, err := email.PlainTextAlert(alert, e.notifier.UiEndpoint, e.mitreStore)
	if err != nil {
		return errors.Wrap(err, "failed to generate email text for alert")
	}

	recipient := e.metadataGetter.GetAnnotationValue(ctx, alert, e.notifier.GetLabelKey(), e.notifier.GetLabelDefault())
	msg := email.Message{
		To:        []string{recipient},
		Subject:   subject,
		Body:      body,
		EmbedLogo: false,
	}

	return e.send(ctx, &msg)
}

func (e *acscsEmail) NetworkPolicyYAMLNotify(ctx context.Context, yaml string, clusterName string) error {
	subject := email.NetworkPolicySubject(clusterName)
	body, err := email.FormatNetworkPolicyYAML(yaml, clusterName)
	if err != nil {
		return errors.Wrap(err, "failed to format network policy message")
	}

	msg := email.Message{
		To:        []string{e.notifier.GetLabelDefault()},
		Subject:   subject,
		Body:      body,
		EmbedLogo: false,
	}

	return e.send(ctx, &msg)
}

func (e *acscsEmail) ReportNotify(ctx context.Context, zippedReportData *bytes.Buffer, recipients []string, subject, messageText string) error {
	// using empty from here because the From header will be set by the managed service
	msg := email.BuildReportMessage(recipients, "", subject, messageText, zippedReportData)
	return e.send(ctx, &msg)
}

func (e *acscsEmail) send(ctx context.Context, msg *email.Message) error {
	apiMsg := AcscsMessage{
		To: msg.To,
		// using ContentBytes instead of Bytes here to allow prepending From and to headers by the
		// ACSCS email service
		RawMessage: msg.ContentBytes(),
	}

	return e.client.SendMessage(ctx, apiMsg)
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
		g, err := newACSCSEmail(notifier, ClientSingleton(), metadatagetter.Singleton(), mitreDS.Singleton(), cryptocodec.Singleton(), cryptoKey)
		return g, err
	})
}
