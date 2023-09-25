package splunk

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/datastore"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/compliance/standards"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/jsonutil"
)

var (
	stateToStringMap = map[storage.ComplianceState]string{
		storage.ComplianceState_COMPLIANCE_STATE_SKIP:    "N/A",
		storage.ComplianceState_COMPLIANCE_STATE_NOTE:    "Info",
		storage.ComplianceState_COMPLIANCE_STATE_SUCCESS: "Pass",
		storage.ComplianceState_COMPLIANCE_STATE_FAILURE: "Fail",
		storage.ComplianceState_COMPLIANCE_STATE_ERROR:   "Error",
	}
)

type splunkComplianceResult struct {
	Standard   string `json:"standard"`
	Cluster    string `json:"cluster"`
	Namespace  string `json:"namespace"`
	ObjectType string `json:"objectType"`
	ObjectName string `json:"objectName"`
	Control    string `json:"control"`
	State      string `json:"state"`
	Evidence   string `json:"evidence"`
}

func stateToString(s storage.ComplianceState) string {
	val, ok := stateToStringMap[s]
	if !ok {
		return "Unknown"
	}
	return val
}

func getMessageLines(evidence []*storage.ComplianceResultValue_Evidence) string {
	lines := make([]string, 0, len(evidence))
	for _, ev := range evidence {
		lines = append(lines, fmt.Sprintf("(%s) %s", stateToString(ev.GetState()), ev.GetMessage()))
	}
	return strings.Join(lines, "\n")
}

// NewComplianceHandler is an HTTP handler that outputs CSV exports of compliance data
func NewComplianceHandler(complianceDS datastore.DataStore) http.HandlerFunc {
	return newComplianceHandler(complianceDS, getClusterIDs)
}

// Internal function that accepts an additional argument, getClusterIDs, that simplifies mocking in tests.
func newComplianceHandler(complianceDS datastore.DataStore, getClusterIDs func(ctx context.Context) ([]string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		arrayWriter := jsonutil.NewJSONArrayWriter(w)
		if err := arrayWriter.Init(); err != nil {
			httputil.WriteError(w, err)
			return
		}

		standardIDs := standards.GetSupportedStandards()
		clusterIDs, err := getClusterIDs(r.Context())
		if err != nil {
			httputil.WriteError(w, err)
			return
		}

		// Iterate over the cluster-standard pairs to minimize memory pressure
		for _, clusterID := range clusterIDs {
			for _, standardID := range standardIDs {
				data, err := complianceDS.GetLatestRunResultsBatch(r.Context(), []string{clusterID}, []string{standardID}, types.RequireMessageStrings)
				if err != nil {
					httputil.WriteError(w, err)
					return
				}
				validResults, _ := datastore.ValidResultsAndSources(data)
				standards := standards.RegistrySingleton()
				for _, d := range validResults {
					controls := make(map[string]*v1.ComplianceControl)
					standardName := d.GetRunMetadata().GetStandardId()
					standard, ok, _ := standards.Standard(standardName)
					if ok {
						standardName = standard.GetMetadata().GetName()
						for _, con := range standard.GetControls() {
							controls[con.GetId()] = con
						}
					}

					for controlID, clusterValue := range d.ClusterResults.GetControlResults() {
						controlName := controlID
						if control, ok := controls[controlID]; ok {
							controlName = control.GetName()
						}
						res := &splunkComplianceResult{
							Standard:   standardName,
							Cluster:    d.GetDomain().GetCluster().GetName(),
							ObjectType: "Cluster",
							ObjectName: d.GetDomain().GetCluster().GetName(),
							Control:    controlName,
							State:      stateToString(clusterValue.OverallState),
							Evidence:   getMessageLines(clusterValue.GetEvidence()),
						}
						if err := arrayWriter.WriteObject(res); err != nil {
							httputil.WriteError(w, err)
							return
						}
					}

					for depKey, depValue := range d.GetDeploymentResults() {
						deployment := d.GetDomain().GetDeployments()[depKey]
						for controlID, result := range depValue.GetControlResults() {
							controlName := controlID
							if control, ok := controls[controlID]; ok {
								controlName = control.GetName()
							}
							res := &splunkComplianceResult{
								Standard:   standardName,
								Cluster:    d.GetDomain().GetCluster().GetName(),
								Namespace:  deployment.GetNamespace(),
								ObjectType: "Deployment",
								ObjectName: deployment.GetName(),
								Control:    controlName,
								State:      stateToString(result.OverallState),
								Evidence:   getMessageLines(result.GetEvidence()),
							}
							if err := arrayWriter.WriteObject(res); err != nil {
								httputil.WriteError(w, err)
								return
							}
						}
					}

					for nodeKey, nodeValue := range d.GetNodeResults() {
						node := d.GetDomain().GetNodes()[nodeKey]
						for controlID, result := range nodeValue.GetControlResults() {
							controlName := controlID
							if control, ok := controls[controlID]; ok {
								controlName = control.GetName()
							}
							res := &splunkComplianceResult{
								Standard:   standardName,
								Cluster:    d.GetDomain().GetCluster().GetName(),
								ObjectType: "Node",
								ObjectName: node.GetName(),
								Control:    controlName,
								State:      stateToString(result.OverallState),
								Evidence:   getMessageLines(result.GetEvidence()),
							}
							if err := arrayWriter.WriteObject(res); err != nil {
								httputil.WriteError(w, err)
								return
							}
						}
					}
					for machineKey, machineValue := range d.GetMachineConfigResults() {
						for controlID, result := range machineValue.GetControlResults() {
							controlName := controlID
							if control, ok := controls[controlID]; ok {
								controlName = control.GetName()
							}
							res := &splunkComplianceResult{
								Standard:   standardName,
								Cluster:    d.GetDomain().GetCluster().GetName(),
								ObjectType: "Machine Config",
								ObjectName: machineKey,
								Control:    controlName,
								State:      stateToString(result.OverallState),
								Evidence:   getMessageLines(result.GetEvidence()),
							}
							if err := arrayWriter.WriteObject(res); err != nil {
								httputil.WriteError(w, err)
								return
							}
						}
					}
				}
			}
		}
		if err := arrayWriter.Finish(); err != nil {
			httputil.WriteError(w, err)
			return
		}
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
