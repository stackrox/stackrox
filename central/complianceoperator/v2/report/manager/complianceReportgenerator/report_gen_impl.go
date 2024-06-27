package complianceReportgenerator

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"
	benchmarksDS "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore"
	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/utils"
	profileDS "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore"
	remediationDS "github.com/stackrox/rox/central/complianceoperator/v2/remediations/datastore"
	complianceRuleDS "github.com/stackrox/rox/central/complianceoperator/v2/rules/datastore"
	scanDS "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/errorhelpers"
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

	reportGenCtx = sac.WithAllAccess(context.Background())

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

const (
	DATA_NOT_AVAILABLE = "Data Not Available"
	NO_REMEDIATION     = "No Remediation Available"
)

type formatSubject struct {
	BrandedPrefix string
	ScanConfig    string
	Profiles      int
}

type complianceReportGeneratorImpl struct {
	checkResultsDS        checkResults.DataStore
	notificationProcessor notifier.Processor
	scanDS                scanDS.DataStore
	profileDS             profileDS.DataStore
	remediationDS         remediationDS.DataStore
	benchmarkDS           benchmarksDS.DataStore
	complianceRuleDS      complianceRuleDS.DataStore
}

// struct which hold all columns of a row
type ResultRow struct {
	ClusterName string
	CheckName   string
	Profile     string
	ControlRef  string
	Description string
	Status      string
	Remediation string
}

type ResultEmail struct {
	ResultCSVs map[string][]*ResultRow // map of cluster id to slice of *resultRow
	TotalPass  int
	TotalFail  int
	TotalMixed int
	Profiles   []string
	Clusters   int
}

func (rg *complianceReportGeneratorImpl) ProcessReportRequest(req *ComplianceReportRequest) {

	log.Infof("Processing report request %s", req)
	data := rg.getDataforReport(req)

	zipData, err := format(data.ResultCSVs)
	if err != nil {
		log.Errorf("Error zipping compliance reports for scan config %s:%s", req.ScanConfigName, err)
		return
	}

	formatEmailBody := &formatBody{
		BrandedPrefix: branding.GetCombinedProductAndShortName(),
		Profile:       strings.Join(req.Profiles, ","),
		Pass:          data.TotalPass,
		Fail:          data.TotalFail,
		Mixed:         data.TotalMixed,
		Cluster:       len(req.ClusterIDs),
	}

	formatEmailSub := &formatSubject{
		BrandedPrefix: branding.GetCombinedProductAndShortName(),
		ScanConfig:    req.ScanConfigName,
		Profiles:      len(req.Profiles),
	}

	log.Infof("Sending email for scan config %s", req.ScanConfigName)
	go rg.sendEmail(req.Ctx, zipData, formatEmailBody, formatEmailSub, req.Notifiers)
}

// getDataforReport returns map of cluster id and slice of ResultRow
func (rg *complianceReportGeneratorImpl) getDataforReport(req *ComplianceReportRequest) *ResultEmail {
	clusters := req.ClusterIDs
	resultsCSV := make(map[string][]*ResultRow)
	resultEmailComplianceReport := &ResultEmail{
		TotalPass:  0,
		TotalMixed: 0,
		TotalFail:  0,
		Clusters:   0,
	}

	for _, clusterID := range clusters {
		parsedQuery := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfig, req.ScanConfigID).
			AddExactMatches(search.ClusterID, clusterID).
			ProtoQuery()
		resultCluster := []*ResultRow{}

		err := rg.checkResultsDS.WalkByQuery(req.Ctx, parsedQuery, func(checkResult *storage.ComplianceOperatorCheckResultV2) error {
			row := &ResultRow{
				ClusterName: checkResult.GetClusterName(),
				CheckName:   checkResult.GetCheckName(),
				Description: checkResult.GetDescription(),
				Status:      checkResult.GetStatus().String(),
			}
			// get profile for the check result
			q := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanRef, checkResult.GetScanRefId()).ProtoQuery()
			profiles, err := rg.profileDS.SearchProfiles(req.Ctx, q)
			if err != nil {
				return err
			}
			if len(profiles) < 1 {
				row.Profile = DATA_NOT_AVAILABLE
				log.Errorf("profile not found for cluster %s and check name %s", clusterID, checkResult.GetCheckName())
			} else {
				row.Profile = fmt.Sprintf("%s %s", profiles[0].GetName(), profiles[0].GetProfileVersion())
			}

			// get remediation for the check result
			q = search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorCheckName, checkResult.GetCheckName()).AddExactMatches(search.ClusterID, checkResult.GetClusterId()).ProtoQuery()
			remediations, err := rg.remediationDS.SearchRemediations(req.Ctx, q)
			if err != nil {
				row.Remediation = DATA_NOT_AVAILABLE
				log.Errorf("remediations not found for cluster %s and check name %s. Error returned %s", clusterID, checkResult.GetCheckName(), err)
			} else if len(remediations) == 0 {
				row.Remediation = NO_REMEDIATION
			} else {
				remediationList := []string{}
				for _, remediation := range remediations {
					remediationList = append(remediationList, remediation.GetName())
				}
				row.Remediation = strings.Join(remediationList, ",")
			}

			// get controls for result and profile
			rules, err := rg.complianceRuleDS.SearchRules(req.Ctx, search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorRuleRef, checkResult.GetRuleRefId()).ProtoQuery())
			if err != nil {
				log.Errorf("Unable to retrieve compliance rule for result %q", checkResult.GetCheckName())
				row.ControlRef = DATA_NOT_AVAILABLE
			} else if len(rules) != 1 {
				// A check result of a cluster maps to a single rule of that same cluster so there should only be 1.
				log.Errorf("Unable to process compliance rule for result %q", checkResult.GetCheckName())
				row.ControlRef = DATA_NOT_AVAILABLE
			} else {
				controls, err := utils.GetControlsForScanResults(req.Ctx, rg.complianceRuleDS, []string{rules[0].GetName()}, profiles[0].GetName(), rg.benchmarkDS)
				if err != nil {
					log.Errorf("Unable to retrieve controls for result %q.Error %s", checkResult.GetCheckName(), err)
					row.ControlRef = DATA_NOT_AVAILABLE
				} else {
					controlsList := []string{}
					for _, ctrl := range controls {
						controlsList = append(controlsList, fmt.Sprintf("%s %s", ctrl.Standard, ctrl.Control))
					}
					row.ControlRef = strings.Join(controlsList, ",")
				}
			}

			resultCluster = append(resultCluster, row)
			if checkResult.GetStatus() == storage.ComplianceOperatorCheckResultV2_FAIL {
				resultEmailComplianceReport.TotalFail += 1
			} else if checkResult.GetStatus() == storage.ComplianceOperatorCheckResultV2_PASS {
				resultEmailComplianceReport.TotalPass += 1
			} else {
				resultEmailComplianceReport.TotalMixed += 1
			}
			return nil
		})
		if err != nil {
			log.Errorf("Data not found for cluster %s", clusterID)
		}

		resultsCSV[clusterID] = resultCluster
	}
	resultEmailComplianceReport.Clusters = len(req.ClusterIDs)
	resultEmailComplianceReport.Profiles = req.Profiles
	resultEmailComplianceReport.ResultCSVs = resultsCSV
	return resultEmailComplianceReport
}

func (rg *complianceReportGeneratorImpl) sendEmail(ctx context.Context, zipData *bytes.Buffer, emailBody *formatBody, formatEmailSub *formatSubject, notifiersList []*storage.NotifierConfiguration) {

	errorList := errorhelpers.NewErrorList("Error sending compliance report email notifications")
	for _, repNotifier := range notifiersList {
		nf := rg.notificationProcessor.GetNotifier(ctx, repNotifier.GetId())
		reportNotifier, ok := nf.(notifiers.ReportNotifier)
		if !ok {
			errorList.AddError(errors.Errorf("incorrect type of notifier %s for compliance report", repNotifier.GetEmailConfig().GetNotifierId()))
			continue
		}

		customBody := repNotifier.GetEmailConfig().GetCustomBody()
		body, err := formatEmailBodywithDetails(defaultEmailBodyTemplate, emailBody)
		if err != nil {
			errorList.AddError(errors.Errorf("Error formatting email body for notifier %s: %s",
				repNotifier.GetEmailConfig().GetNotifierId(), err))
		}
		if customBody != "" {
			body = customBody
		}

		customSubject := repNotifier.GetEmailConfig().GetCustomSubject()
		emailSubject, err := formatEmailSubjectwithDetails(defaultSubjectTemplate, formatEmailSub)
		if err != nil {
			errorList.AddError(errors.Errorf("Error formatting email subject for notifier %s: %s",
				repNotifier.GetEmailConfig().GetNotifierId(), err))
		}
		if customSubject != "" {
			emailSubject = customSubject
		}
		err = retryableSendReportResults(reportNotifier, repNotifier.GetEmailConfig().GetMailingLists(),
			zipData, emailSubject, body)
		if err != nil {
			errorList.AddError(errors.Errorf("Error sending compliance report email for notifier %s: %s",
				repNotifier.GetEmailConfig().GetNotifierId(), err))
		}
	}

	if !errorList.Empty() {
		log.Errorf("Error in sending email to notifiers %s", errorList)
	}
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

func format(results map[string][]*ResultRow) (*bytes.Buffer, error) {
	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)
	for clusterID, res := range results {
		fileName := fmt.Sprintf("cluster_%s.csv", clusterID)
		err := createCSVInZip(zipWriter, fileName, res)
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

func createCSVInZip(zipWriter *zip.Writer, filename string, res []*ResultRow) error {
	w, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}

	csvWriter := csv.NewGenericWriter(csvHeader, true)
	if len(res) != 0 {
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
	} else {
		csvWriter.AddValue([]string{"Data not found for the cluster"})
	}
	return csvWriter.WriteCSV(w)
}
