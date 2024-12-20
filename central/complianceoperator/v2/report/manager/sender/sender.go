package sender

import (
	"bytes"
	"context"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/notifier"
	reportNotifiers "github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/templates"
)

const (
	defaultEmailBodyTemplate = "{{.BrandedPrefix}} has scanned your clusters for compliance with the profiles in your scan configuration." +
		"The attached report lists the checks performed and provides corresponding details to help with remediation. \n" +
		"Profiles:{{.Profile}} |\n" +
		"Passing:{{.Pass}} checks |\n" +
		"Failing:{{.Fail}} checks |\n" +
		"Mixed:{{.Mixed}} checks |\n" +
		"Clusters: {{.Cluster}} scanned"

	defaultSubjectTemplate = "{{.BrandedPrefix}} Compliance Report For {{.ScanConfig}} with {{.Profiles}} Profiles"

	defaultNumberOfRetries = 3
)

type ReportSender struct {
	notifierProcessor notifier.Processor
	numRetries        int
}

func NewReportSender(processor notifier.Processor, numRetries int) *ReportSender {
	if numRetries < 1 {
		numRetries = defaultNumberOfRetries
	}
	return &ReportSender{
		notifierProcessor: processor,
		numRetries:        numRetries,
	}
}

type formatBody struct {
	BrandedPrefix string
	Profile       string
	Pass          int
	Fail          int
	Mixed         int
	Cluster       int
}

type formatSubject struct {
	BrandedPrefix string
	ScanConfig    string
	Profiles      int
}

func (s *ReportSender) SendEmail(ctx context.Context, scanConfigName string, data *bytes.Buffer, results *report.Results, notifiers []*storage.NotifierConfiguration) <-chan error {
	formatEmailBody := &formatBody{
		BrandedPrefix: branding.GetCombinedProductAndShortName(),
		Profile:       strings.Join(results.Profiles, ","),
		Pass:          results.TotalPass,
		Fail:          results.TotalFail,
		Mixed:         results.TotalMixed,
		Cluster:       results.Clusters,
	}
	formatEmailSubject := &formatSubject{
		BrandedPrefix: branding.GetCombinedProductAndShortName(),
		ScanConfig:    scanConfigName,
		Profiles:      len(results.Profiles),
	}
	errC := make(chan error)
	go s.send(ctx, data, formatEmailSubject, formatEmailBody, notifiers, scanConfigName, errC)
	return errC
}

func (s *ReportSender) send(ctx context.Context, data *bytes.Buffer, subject *formatSubject, body *formatBody, notifiers []*storage.NotifierConfiguration, reportName string, errC chan error) {
	defer close(errC)
	errorList := errorhelpers.NewErrorList("Error sending compliance report email notifications")
	for _, notifierConfig := range notifiers {
		notifierID := notifierConfig.GetEmailConfig().GetNotifierId()
		reportNotifier, ok := s.notifierProcessor.GetNotifier(ctx, notifierConfig.GetId()).(reportNotifiers.ReportNotifier)
		if !ok {
			errorList.AddError(errors.Errorf("incorrect type of notifier %s for compliance report", notifierID))
			continue
		}

		emailSubject, emailBody, err := s.formatEmail(subject, body, notifierConfig)
		if err != nil {
			errorList.AddError(err)
		}

		err = s.sendResultsWithRetry(ctx, reportNotifier, notifierConfig.GetEmailConfig().GetMailingLists(), data, emailSubject, emailBody, reportName)
		if err != nil {
			errorList.AddError(errors.Wrapf(err, "unable to send compliance report email for notifier %s", notifierID))
		}
	}

	errC <- errorList.ToError()
}

func (s *ReportSender) formatEmail(subject *formatSubject, body *formatBody, notifierConfig *storage.NotifierConfiguration) (string, string, error) {
	errorList := errorhelpers.NewErrorList("Error formatting the email")

	customSubject := notifierConfig.GetEmailConfig().GetCustomSubject()
	emailSubject, err := formatWithDetails("emailSubject", defaultSubjectTemplate, subject)
	if err != nil {
		errorList.AddError(errors.Wrapf(err, "unable to format email subject for notifier %s", notifierConfig.GetEmailConfig().GetNotifierId()))
	}
	if customSubject != "" {
		emailSubject = customSubject
	}

	customBody := notifierConfig.GetEmailConfig().GetCustomBody()
	emailBody, err := formatWithDetails("emailBody", defaultEmailBodyTemplate, body)
	if err != nil {
		errorList.AddError(errors.Wrapf(err, "unable to format email body for notifier %s", notifierConfig.GetEmailConfig().GetNotifierId()))
	}
	if customBody != "" {
		emailBody = customBody
	}

	return emailSubject, emailBody, errorList.ToError()
}

func (s *ReportSender) sendResultsWithRetry(ctx context.Context, reportNotifier reportNotifiers.ReportNotifier, mailingList []string, data *bytes.Buffer, subject, body, reportName string) error {
	return retry.WithRetry(func() error {
		return reportNotifier.ReportNotify(ctx, data, mailingList, subject, body, reportName)
	},
		retry.OnlyRetryableErrors(),
		retry.Tries(s.numRetries),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Second)
		}),
	)
}

func formatWithDetails(templateName string, format string, data any) (string, error) {
	tmpl, err := template.New(templateName).Parse(format)
	if err != nil {
		return "", err
	}
	return templates.ExecuteToString(tmpl, data)
}
