package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

type options struct {
	format                  string
	clusterIDs, standardIDs []string
}

func idfilter(ids []string) func(string) bool {
	if len(ids) == 0 {
		return func(string) bool { return true }
	}
	return func(s string) bool {
		for _, v := range ids {
			if s == v {
				return true
			}
		}
		return false
	}
}

func (o options) clusterIDFilter() func(string) bool {
	return idfilter(o.clusterIDs)
}

func (o options) standardIDFilter() func(string) bool {
	return idfilter(o.standardIDs)
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

func writeErr(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	fmt.Fprint(w, err)
}

type complianceRow struct {
	standardID         string
	controlName        string
	controlDescription string
	clusterName        string
	objectType         string
	objectName         string
	objectNamespace    string
	value              *storage.ComplianceResultValue
	runTimestamp       string
}

type csvResults struct {
	header []string
	values [][]string
}

func (c *csvResults) write(writer *csv.Writer) {
	sort.Slice(c.values, func(i, j int) bool {
		first, second := c.values[i], c.values[j]
		for len(first) > 0 {
			// first has more values, so greater
			if len(second) == 0 {
				return false
			}
			if first[0] < second[0] {
				return true
			}
			if first[0] > second[0] {
				return false
			}
			first = first[1:]
			second = second[1:]
		}
		// second has more values, so first is lesser
		return len(second) > 0
	})
	writer.Write(c.header)
	for _, v := range c.values {
		writer.Write(v)
	}
}

func getControlDescription(control *v1.ComplianceControl) string {
	des := control.GetDescription()
	if len(des) > 60 {
		idx := strings.IndexAny(des, ".:")
		if idx > 20 {
			des = des[:idx] + " \u2026"
		}
	}
	return des
}

func (c *csvResults) addAll(row complianceRow, controls map[string]*v1.ComplianceControl, values map[string]*storage.ComplianceResultValue) {
	for controlID, result := range values {
		controlName := controlID
		controlDescription := "-"
		if control, ok := controls[controlID]; ok {
			controlName = control.GetName()
			controlDescription = getControlDescription(control)
		}
		valueRow := row
		valueRow.controlName = controlName
		valueRow.value = result
		valueRow.controlDescription = controlDescription
		c.addRow(valueRow)
	}
}

var (
	stateNames = map[storage.ComplianceState]string{
		storage.ComplianceState_COMPLIANCE_STATE_ERROR:   "Error",
		storage.ComplianceState_COMPLIANCE_STATE_FAILURE: "Fail",
		storage.ComplianceState_COMPLIANCE_STATE_SUCCESS: "Pass",
		storage.ComplianceState_COMPLIANCE_STATE_SKIP:    "-",
	}
)

func stateToString(s storage.ComplianceState) string {
	val, ok := stateNames[s]
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
	}
	for _, ev := range row.value.GetEvidence() {
		value2 := append(value, stateToString(ev.GetState()), ev.GetMessage(), row.runTimestamp)
		c.values = append(c.values, value2)
	}
}

func fromTS(timestamp *types.Timestamp) string {
	if timestamp == nil {
		return "-"
	}
	ts, err := types.TimestampFromProto(timestamp)
	if err != nil {
		return "!err"
	}
	return ts.Format(time.RFC3339)
}

// CSVHandler is an HTTP handler that outputs CSV exports of compliance data
func CSVHandler() http.HandlerFunc {
	complianceStore := store.Singleton()
	return func(w http.ResponseWriter, r *http.Request) {
		options, err := parseOptions(r)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		data, err := complianceStore.GetLatestRunResultsFiltered(options.clusterIDFilter(), options.standardIDFilter())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		validResults, _ := store.ValidResultsAndSources(data)
		var output csvResults
		output.header = []string{"standard", "cluster", "namespace", "type", "object", "control", "description", "state", "evidence", "timestamp"}
		standards := standards.RegistrySingleton()
		for _, d := range validResults {
			controls := make(map[string]*v1.ComplianceControl)
			standardName := d.GetRunMetadata().GetStandardId()
			timestamp := fromTS(d.GetRunMetadata().GetFinishTimestamp())
			standard, ok, _ := standards.Standard(standardName)
			if ok {
				standardName = standard.GetMetadata().GetName()
				for _, con := range standard.GetControls() {
					controls[fmt.Sprintf("%s:%s", standard.GetMetadata().GetId(), con.GetId())] = con
				}
			}
			dataRow := complianceRow{
				standardID:   standardName,
				clusterName:  d.GetDomain().GetCluster().GetName(),
				runTimestamp: timestamp,
			}
			for dk, dv := range d.GetDeploymentResults() {
				deploymentRow := dataRow
				deployment := d.GetDomain().GetDeployments()[dk]
				deploymentRow.objectType = deployment.GetType()
				deploymentRow.objectNamespace = deployment.GetNamespace()
				deploymentRow.objectName = deployment.GetName()
				output.addAll(deploymentRow, controls, dv.GetControlResults())
			}
			dataRow.objectNamespace = ""
			for node, values := range d.GetNodeResults() {
				nodeRow := dataRow
				node := d.GetDomain().GetNodes()[node]
				nodeRow.objectType = "node"
				nodeRow.objectName = node.GetName()
				output.addAll(nodeRow, controls, values.GetControlResults())
			}
			dataRow.objectType = "cluster"
			dataRow.objectName = dataRow.clusterName
			output.addAll(dataRow, controls, d.GetClusterResults().GetControlResults())
		}
		w.Header().Set("Content-Type", `text/csv; charset="utf-8"`)
		w.Header().Set("Content-Disposition", `attachment; filename="compliance_export.csv"`)
		w.WriteHeader(http.StatusOK)
		cw := csv.NewWriter(w)
		cw.UseCRLF = true
		output.write(cw)
		cw.Flush()
	}
}
