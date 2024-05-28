package complianceReportgenerator

import (
	"archive/zip"
	"bytes"
	"context"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
<<<<<<< HEAD
=======
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
>>>>>>> 6faeddcd64 (Added test file)
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/errorhelpers"
<<<<<<< HEAD
=======
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
>>>>>>> 6faeddcd64 (Added test file)
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/templates"
)

var (
	log = logging.LoggerForModule()

<<<<<<< HEAD
	reportGenCtx = sac.WithAllAccess(context.Background())
=======
	reportGenCtx = resolvers.SetAuthorizerOverride(loaders.WithLoaderContext(sac.WithAllAccess(context.Background())), allow.Anonymous())
>>>>>>> 6faeddcd64 (Added test file)

	csvHeader = []string{
		"Control Reference",
		"Check(CCR)",
		"Profile(version)",
		"Check Description",
		"Cluster",
		"Status",
		"Remediation",
	}
)

const (
	maxNumberProfilesinSubject = 4
)

type formatBody struct {
<<<<<<< HEAD
	BrandedPrefix string
	Profile       string
	Pass          int
	Fail          int
	Mixed         int
	Cluster       int
=======
	BrandedPrefix    string
	Profile          string
	Pass             int
	Fail             int
	Mixed            int
	Cluster          int
	ComplianceStatus string
>>>>>>> 6faeddcd64 (Added test file)
}

type formatSubject struct {
	BrandedPrefix string
	ScanConfig    string
<<<<<<< HEAD
	Profiles      int
=======
	Profiles      string
>>>>>>> 6faeddcd64 (Added test file)
}

type complianceReportGeneratorImpl struct {
	checkResultsDS        checkResults.DataStore
	notificationProcessor notifier.Processor
}

// struct which hold all columns of a row
<<<<<<< HEAD
type ResultRow struct {
=======
type resultRow struct {
>>>>>>> 6faeddcd64 (Added test file)
	ClusterName string
	CheckName   string
	Profile     string
	ControlRef  string
	Description string
	Status      string
	Remediation string
}

<<<<<<< HEAD
type ResultEmail struct {
	ResultCSVs map[string][]*ResultRow // map of cluster id to slice of *resultRow
	TotalPass  int
	TotalFail  int
	TotalMixed int
	Profiles   []string
	Clusters   int
}

func (rg *complianceReportGeneratorImpl) ProcessReportRequest(req *ComplianceReportRequest) {

	data, err := rg.getDataForReport(req)
	if err != nil {
		log.Errorf("Error getting report data for scan config %s", req.ScanConfigName)
		return
	}
	if data == nil {
		log.Errorf("Error getting report data for scan config %s", req.ScanConfigName)
		return
	}

	zipData, err := format(data.ResultCSVs)
	if err != nil {
		log.Errorf("Error zipping compliance reports for scan config %s", req.ScanConfigName)
		return
	}
	profiles := strings.Join(req.Profiles, ", ")
	formatEmailBody := &formatBody{
		BrandedPrefix: branding.GetCombinedProductAndShortName(),
		Profile:       profiles,
		Pass:          data.TotalPass,
		Fail:          data.TotalFail,
		Mixed:         data.TotalMixed,
		Cluster:       len(req.ClusterIDs),
	}

	if len(profiles) > maxNumberProfilesinSubject {
		profiles = strings.Join(req.Profiles[0:maxNumberProfilesinSubject], ", ")
=======
type resultEmail struct {
	resultCSVs map[string][]*resultRow //map of cluster id to slice of *resultRow
	totalPass  int
	totalFail  int
	totalMixed int
	profiles   []string
	clusters   int
}

func (rg *complianceReportGeneratorImpl) ProcessReportRequest(ctx context.Context, req *ComplianceReportRequest) error {

	data, err := rg.getDataforReport(req, ctx)
	if err != nil {
		return err
	}

	zipData, err := rg.Format(data.resultCSVs)
	if err != nil {
		return err
	}
	profiles := strings.Join(req.profiles, ", ")
	formatEmailBody := &formatBody{
		BrandedPrefix: branding.GetCombinedProductAndShortName(),
		Profile:       profiles,
		Pass:          data.totalPass,
		Fail:          data.totalFail,
		Mixed:         data.totalMixed,
		Cluster:       len(req.clusterIDs),
	}
	formatEmailBody.ComplianceStatus = "non compliant"
	if data.totalFail == 0 {
		formatEmailBody.ComplianceStatus = "compliant"
	}

	if len(profiles) > maxNumberProfilesinSubject {
		profiles = strings.Join(req.profiles[0:maxNumberProfilesinSubject], ", ")
>>>>>>> 6faeddcd64 (Added test file)
		profiles += "..."
	}

	formatEmailSub := &formatSubject{
		BrandedPrefix: branding.GetCombinedProductAndShortName(),
<<<<<<< HEAD
		ScanConfig:    req.ScanConfigName,
		Profiles:      len(profiles),
	}

	log.Infof("Sending email for scan config %s", req.ScanConfigName)
	go rg.sendEmail(req.Ctx, zipData, formatEmailBody, formatEmailSub, req.Notifiers)
}

// getDataForReport returns map of cluster id and
func (rg *complianceReportGeneratorImpl) getDataForReport(req *ComplianceReportRequest) (*ResultEmail, error) {
=======
		ScanConfig:    req.scanConfigName,
		Profiles:      profiles,
	}

	return rg.sendEmail(zipData, formatEmailBody, formatEmailSub, req.notifiers, ctx)
}

// getDataforReport returns map of cluster id and
func (rg *complianceReportGeneratorImpl) getDataforReport(req *ComplianceReportRequest, ctx context.Context) (*resultEmail, error) {
>>>>>>> 6faeddcd64 (Added test file)
	// TODO ROX-24356: Implement query to get checkresults data to generate cvs for compliance reporting

	return nil, nil
}

<<<<<<< HEAD
func (rg *complianceReportGeneratorImpl) sendEmail(ctx context.Context, zipData *bytes.Buffer, emailBody *formatBody, formatEmailSub *formatSubject, notifiersList []*storage.NotifierConfiguration) {
=======
func (rg *complianceReportGeneratorImpl) sendEmail(zipData *bytes.Buffer, emailBody *formatBody, formatEmailSub *formatSubject, notifiersList []*storage.NotifierConfiguration, ctx context.Context) error {
>>>>>>> 6faeddcd64 (Added test file)

	errorList := errorhelpers.NewErrorList("Error sending compliance report email notifications")
	for _, notifier := range notifiersList {
		nf := rg.notificationProcessor.GetNotifier(ctx, notifier.GetEmailConfig().GetNotifierId())
		reportNotifier, ok := nf.(notifiers.ReportNotifier)
		if !ok {
			errorList.AddError(errors.Errorf("incorrect type of notifier '%s' for compliance report", notifier.GetEmailConfig().GetNotifierId()))
			continue
		}

		customBody := notifier.GetEmailConfig().GetCustomBody()
		emailBody, err := formatEmailBodywithDetails(defaultEmailBodyTemplate, emailBody)
		if err != nil {
			errorList.AddError(errors.Errorf("Error sending compliance report email for notifier '%s': %s",
				notifier.GetEmailConfig().GetNotifierId(), err))
		}
		if customBody != "" {
			emailBody = customBody
		}

		customSubject := notifier.GetEmailConfig().GetCustomSubject()
		emailSubject, err := formatEmailSubjectwithDetails(defaultSubjectTemplate, formatEmailSub)
		if err != nil {
			errorList.AddError(errors.Errorf("Error sending compliance report email for notifier '%s': %s",
				notifier.GetEmailConfig().GetNotifierId(), err))
		}
		if customSubject != "" {
			emailSubject = customSubject
		}
		err = retryableSendReportResults(reportNotifier, notifier.GetEmailConfig().GetMailingLists(),
			zipData, emailSubject, emailBody)
		if err != nil {
			errorList.AddError(errors.Errorf("Error sending compliance report email for notifier '%s': %s",
				notifier.GetEmailConfig().GetNotifierId(), err))
		}
	}

<<<<<<< HEAD
	if !errorList.Empty() {
		log.Errorf("Error in sending email to notifiers %s", errorList)
	}
=======
	return errorList
>>>>>>> 6faeddcd64 (Added test file)
}

func formatEmailSubjectwithDetails(subject string, data *formatSubject) (string, error) {
	tmpl, err := template.New("emailSubject").Parse(subject)
	if err != nil {
		return "", err
	}
	return templates.ExecuteToString(tmpl, data)
}

func formatEmailBodywithDetails(subject string, data *formatBody) (string, error) {
	tmpl, err := template.New("emailBody").Parse(subject)
	if err != nil {
		return "", err
	}
	return templates.ExecuteToString(tmpl, data)
}

func retryableSendReportResults(reportNotifier notifiers.ReportNotifier, mailingList []string,
	zippedCSVData *bytes.Buffer, emailSubject, emailBody string) error {
	return retry.WithRetry(func() error {
		return reportNotifier.ReportNotify(reportGenCtx, zippedCSVData, mailingList, emailSubject, emailBody)
	},
		retry.OnlyRetryableErrors(),
		retry.Tries(3),
		retry.BetweenAttempts(func(previousAttempt int) {
			wait := time.Duration(previousAttempt * previousAttempt * 100)
			time.Sleep(wait * time.Millisecond)
		}),
	)
}

<<<<<<< HEAD
func format(results map[string][]*ResultRow) (*bytes.Buffer, error) {
=======
func (rg *complianceReportGeneratorImpl) Format(results map[string][]*resultRow) (*bytes.Buffer, error) {
>>>>>>> 6faeddcd64 (Added test file)
	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)
	for cluster, res := range results {
		err := createCSVInZip(zipWriter, cluster, res)
		if err != nil {
			return nil, errors.Wrap(err, "error creating csv report")
		}
	}

	err := zipWriter.Close()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create a zip file of the vuln report")
	}
	return &zipBuf, nil
}

<<<<<<< HEAD
func createCSVInZip(zipWriter *zip.Writer, filename string, res []*ResultRow) error {
=======
func createCSVInZip(zipWriter *zip.Writer, filename string, res []*resultRow) error {
>>>>>>> 6faeddcd64 (Added test file)
	w, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}

	csvWriter := csv.NewGenericWriter(csvHeader, true)
	for _, checkRes := range res {
		record := []string{
			checkRes.ControlRef,
			checkRes.CheckName,
			checkRes.Profile,
			checkRes.Description,
			checkRes.ClusterName,
			checkRes.Status,
			checkRes.Remediation,
		}
		csvWriter.AddValue(record)
	}

	return csvWriter.WriteCSV(w)
}
