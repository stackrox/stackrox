package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/datastore"
	complianceDSTypes "github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/compliance/standards"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/csv"
)

type options struct {
	format                  string
	clusterIDs, standardIDs []string
}

func parseOptions(r *http.Request) (options options, err error) {
	err = r.ParseForm()
	if err != nil {
		return
	}
	options.format = r.Form.Get("format")
	if options.format == "" {
		options.format = "list"
	}
	if options.format != "list" {
		err = fmt.Errorf("invalid value for option %q", "format")
		return
	}
	options.clusterIDs = r.Form["clusterId"]
	options.standardIDs = r.Form["standardId"]
	return
}

type complianceRow struct {
	standardID         string
	controlName        string
	controlDescription string
	clusterName        string
	objectType         string
	objectName         string
	objectNamespace    string
	result             *storage.ComplianceResultValue
	runTimestamp       string
}

type csvResults struct {
	*csv.GenericWriter
}

func newCSVResults(header []string) csvResults {
	return csvResults{
		GenericWriter: csv.NewGenericWriter(header, true),
	}
}

func (c *csvResults) addAll(row complianceRow, controls map[string]*v1.ComplianceControl, values map[string]*storage.ComplianceResultValue) {
	for controlID, result := range values {
		controlName := controlID
		controlDescription := "N/A"
		if control, ok := controls[controlID]; ok {
			controlName = control.GetName()
			controlDescription = control.GetDescription()
		}
		valueRow := row
		valueRow.result = result
		valueRow.controlName = fmt.Sprintf(`=("%s")`, controlName) // avoid excel parsing as a number
		valueRow.controlDescription = controlDescription
		c.addRow(valueRow)
	}
}

var (
	stateToStringMap = map[storage.ComplianceState]string{
		storage.ComplianceState_COMPLIANCE_STATE_SKIP:    "N/A",
		storage.ComplianceState_COMPLIANCE_STATE_NOTE:    "Info",
		storage.ComplianceState_COMPLIANCE_STATE_SUCCESS: "Pass",
		storage.ComplianceState_COMPLIANCE_STATE_FAILURE: "Fail",
		storage.ComplianceState_COMPLIANCE_STATE_ERROR:   "Error",
	}
)

func stateToString(s storage.ComplianceState) string {
	val, ok := stateToStringMap[s]
	if !ok {
		return "Unknown"
	}
	return val
}

func (c *csvResults) addRow(row complianceRow) {
	// standard, cluster, type, namespace, object, control, state, evidence
	value := []string{
		row.standardID,
		row.clusterName,
		row.objectNamespace,
		row.objectType,
		row.objectName,
		row.controlName,
		row.controlDescription,
		stateToString(row.result.OverallState),
	}

	lines := make([]string, 0, len(row.result.GetEvidence()))
	for i, ev := range row.result.GetEvidence() {
		lines = append(lines, fmt.Sprintf("%d. (%s) %s", i+1, stateToString(ev.GetState()), ev.GetMessage()))
	}
	combinedEvidence := strings.Join(lines, "\n")

	value = append(value, combinedEvidence, row.runTimestamp)
	c.AddValue(value)
}

// CSVHandler is an HTTP handler that outputs CSV exports of compliance data
func CSVHandler() http.HandlerFunc {
	complianceDS := datastore.Singleton()
	return func(w http.ResponseWriter, r *http.Request) {
		options, err := parseOptions(r)
		if err != nil {
			csv.WriteErrorWithCode(w, http.StatusBadRequest, err)
			return
		}

		// If the request to export csv does not include any standards filters, add supported standards filter
		if len(options.standardIDs) == 0 {
			options.standardIDs = standards.GetSupportedStandards()
		} else {
			var unsupported []string
			options.standardIDs, unsupported = standards.FilterSupported(options.standardIDs)
			if len(unsupported) > 0 {
				csv.WriteError(w, standards.UnSupportedStandardsErr(unsupported...))
				return
			}
		}

		if len(options.clusterIDs) == 0 {
			clusterIDs, err := getClusterIDs(r.Context())
			if err != nil {
				csv.WriteErrorWithCode(w, http.StatusInternalServerError, err)
				return
			}
			options.clusterIDs = clusterIDs
		}

		data, err := complianceDS.GetLatestRunResultsBatch(r.Context(), options.clusterIDs, options.standardIDs, complianceDSTypes.WithMessageStrings)
		if err != nil {
			csv.WriteErrorWithCode(w, http.StatusInternalServerError, err)
			return
		}
		validResults, _ := datastore.ValidResultsAndSources(data)
		output := newCSVResults([]string{"Standard", "Cluster", "Namespace", "Object Type", "Object Name", "Control", "Control Description", "State", "Evidence", "Assessment Time"})
		standards := standards.RegistrySingleton()
		for _, d := range validResults {
			controls := make(map[string]*v1.ComplianceControl)
			standardName := d.GetRunMetadata().GetStandardId()
			timestamp := csv.FromTimestamp(d.GetRunMetadata().GetFinishTimestamp())
			standard, ok, _ := standards.Standard(standardName)
			if ok {
				standardName = standard.GetMetadata().GetName()
				for _, con := range standard.GetControls() {
					controls[con.GetId()] = con
				}
			}
			dataRow := complianceRow{
				standardID:   standardName,
				clusterName:  d.GetDomain().GetCluster().GetName(),
				runTimestamp: timestamp,
			}
			for depKey, depValue := range d.GetDeploymentResults() {
				deploymentRow := dataRow
				deployment := d.GetDomain().GetDeployments()[depKey]
				deploymentRow.objectType = deployment.GetType()
				deploymentRow.objectNamespace = deployment.GetNamespace()
				deploymentRow.objectName = deployment.GetName()
				output.addAll(deploymentRow, controls, depValue.GetControlResults())
			}
			dataRow.objectNamespace = ""
			for nodeKey, nodeValue := range d.GetNodeResults() {
				nodeRow := dataRow
				node := d.GetDomain().GetNodes()[nodeKey]
				nodeRow.objectType = "Node"
				nodeRow.objectName = node.GetName()
				output.addAll(nodeRow, controls, nodeValue.GetControlResults())
			}
			dataRow.objectType = "Cluster"
			dataRow.objectName = dataRow.clusterName
			output.addAll(dataRow, controls, d.GetClusterResults().GetControlResults())

			dataRow.objectType = "MachineConfig"
			for mcKey, mcValue := range d.GetMachineConfigResults() {
				mcRow := dataRow
				mcRow.objectName = mcKey
				output.addAll(mcRow, controls, mcValue.GetControlResults())
			}
		}
		output.Write(w, "compliance_export")
	}
}

func getClusterIDs(ctx context.Context) ([]string, error) {
	clusterDS := clusterDatastore.Singleton()

	clusters, err := clusterDS.GetClusters(ctx)
	if err != nil {
		return nil, err
	}

	clusterIDs := make([]string, len(clusters))
	for i, cluster := range clusters {
		clusterIDs[i] = cluster.GetId()
	}
	return clusterIDs, nil
}
