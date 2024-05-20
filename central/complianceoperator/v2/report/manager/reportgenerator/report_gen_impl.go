package reportgenerator

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/pkg/errors"
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifier"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/templates"
)

var (
	log = logging.LoggerForModule()

	reportGenCtx = resolvers.SetAuthorizerOverride(loaders.WithLoaderContext(sac.WithAllAccess(context.Background())), allow.Anonymous())

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
	Profiles      string
}

type reportGeneratorImpl struct {
	checkResultsDS        checkResults.DataStore
	notificationProcessor notifier.Processor
}

type resultRow struct {
	ClusterName string
	CheckName   string
	Profile     string
	ControlRef  string
	Description string
	Status      string
	Remediation string
}

type resultEmail struct {
	resultCSVs map[string][]*resultRow
	totalPass  int
	totalFail  int
	totalMixed int
	profiles   []string
	clusters   int
}

func (rg *reportGeneratorImpl) ProcessReportRequest(ctx context.Context, req *ComplianceReportRequest) error {
	//query compliance data
	// Add the scan config name as an exact match
	clusters := req.clusterIDs

	var resultsCSV map[string][]*resultRow

	resultEmailComplianceReport := &resultEmail{
		totalPass:  0,
		totalMixed: 0,
		totalFail:  0,
		clusters:   0,
	}

	for _, clusterID := range clusters {

		parsedQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfig, req.scanConfigID).
			AddExactMatches(search.ClusterID, clusterID).
			ProtoQuery()
		results, err := rg.checkResultsDS.SearchComplianceCheckResults(ctx, parsedQuery)
		if err != nil {
			return err
		}

		resultCluster := []*resultRow{}

		for _, resultCheck := range results {
			row := &resultRow{
				ClusterName: resultCheck.GetClusterName(),
				CheckName:   resultCheck.GetCheckName(),
				Description: resultCheck.GetDescription(),
				Status:      resultCheck.GetStatus().String(),
				Remediation: resultCheck.GetInstructions(),
				Profile:     "Profile",
				ControlRef:  "Control Ref",
			}
			resultCluster = append(resultCluster, row)
			if resultCheck.GetStatus() == storage.ComplianceOperatorCheckResultV2_PASS {
				resultEmailComplianceReport.totalPass += 1
			} else if resultCheck.GetStatus() == storage.ComplianceOperatorCheckResultV2_FAIL {
				resultEmailComplianceReport.totalFail += 1
			} else {
				resultEmailComplianceReport.totalMixed += 1
			}
		}
		resultsCSV[clusterID] = resultCluster
	}

	resultEmailComplianceReport.clusters = len(req.clusterIDs)
	resultEmailComplianceReport.profiles = req.profiles
	resultEmailComplianceReport.resultCSVs = resultsCSV

	zipData, err := rg.Format(resultEmailComplianceReport.resultCSVs)
	if err != nil {
		return err
	}
	var profiles string
	for index, profile := range req.profiles {
		if index == len(req.profiles)-1 {
			profiles += fmt.Sprintf("%s", profile)
			break
		}
		profiles += fmt.Sprintf("%s,", profile)
	}
	formatEmailBody := &formatBody{
		BrandedPrefix: branding.GetCombinedProductAndShortName(),
		Profile:       profiles,
		Pass:          resultEmailComplianceReport.totalPass,
		Fail:          resultEmailComplianceReport.totalFail,
		Mixed:         resultEmailComplianceReport.totalMixed,
		Cluster:       len(req.clusterIDs),
	}

	formatEmailSub := &formatSubject{
		BrandedPrefix: branding.GetCombinedProductAndShortName(),
		ScanConfig:    req.scanConfigName,
		Profiles:      profiles,
	}

	return rg.sendEmail(zipData, formatEmailBody, formatEmailSub, req.notifiers, ctx)
}

func (rg *reportGeneratorImpl) sendEmail(zipData *bytes.Buffer, emailBody *formatBody, formatEmailSub *formatSubject, notifiersList []*storage.NotifierConfiguration, ctx context.Context) error {

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
		err = rg.retryableSendReportResults(reportNotifier, notifier.GetEmailConfig().GetMailingLists(),
			zipData, emailSubject, emailBody)
		if err != nil {
			errorList.AddError(errors.Errorf("Error sending compliance report email for notifier '%s': %s",
				notifier.GetEmailConfig().GetNotifierId(), err))
		}
	}

	return errorList
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

func (rg *reportGeneratorImpl) retryableSendReportResults(reportNotifier notifiers.ReportNotifier, mailingList []string,
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

func (rg *reportGeneratorImpl) Format(results map[string][]*resultRow) (*bytes.Buffer, error) {
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

func createCSVInZip(zipWriter *zip.Writer, filename string, res []*resultRow) error {
	w, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}

	//csvWriter := csv.NewWriter(w)
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
