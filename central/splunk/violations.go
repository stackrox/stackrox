package splunk

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/generated/api/integrations"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"go.uber.org/zap"
)

const (
	checkpointQueryParam    = "from_checkpoint"
	fallbackCheckpointValue = "2000-01-01T00:00:00.000Z"
)

var (
	// fallbackCheckpoint is a timestamp to use when checkpointQueryParam is not present in the query or can't be parsed.
	fallbackCheckpoint, fallbackCheckpointTime = func() (*types.Timestamp, time.Time) {
		ts, t, err := parseTimestamp(fallbackCheckpointValue)
		utils.Must(err)
		return ts, t
	}()
)

func parseTimestamp(timeStr string) (*types.Timestamp, time.Time, error) {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return nil, time.Time{}, err
	}
	ts, err := types.TimestampProto(t)
	if err != nil {
		return nil, time.Time{}, err
	}
	return ts, t, nil
}

// NewViolationsHandler provides violations data to Splunk on HTTP requests.
func NewViolationsHandler(alertDS datastore.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		res, err := getViolationsResponse(alertDS, r)
		if err != nil {
			msg := fmt.Sprintf("Error handling Splunk violations request: %s", err)
			log.Warn(msg)
			httputil.WriteError(w, err)
			return
		}
		err = (&jsonpb.Marshaler{}).Marshal(w, res)
		if err != nil {
			log.Warn("Error writing violations response: ", err)
			panic(http.ErrAbortHandler)
		}
	}
}

func getViolationsResponse(alertDS datastore.DataStore, r *http.Request) (*integrations.SplunkViolationsResponse, error) {
	fromTimestamp, fromTime, err := getCheckpointValue(r)
	if err != nil {
		return nil, err
	}

	// TODO(ROX-6689): pagination
	alerts, err := queryAlerts(r.Context(), alertDS, fromTime)
	if err != nil {
		return nil, err
	}

	response := integrations.SplunkViolationsResponse{
		// Checkpoint should remain the same as before if there are no violations.
		// Strictly speaking, we don't have to report it because Splunk remembers the last non-empty checkpoint, but
		// that's just more consistent to always return a value here.
		NewCheckpoint: fromTimestamp,
	}

	for _, alert := range alerts {
		violations, err := extractViolations(alert, fromTimestamp)
		if err != nil {
			return nil, err
		}
		response.Violations = append(response.Violations, violations...)
	}

	sort.Slice(response.Violations, func(i, j int) bool {
		return response.Violations[i].GetViolationInfo().GetViolationTime().Compare(response.Violations[j].GetViolationInfo().GetViolationTime()) < 0
	})

	if len(response.Violations) > 0 {
		// NewCheckpoint must be max of ViolationTimes so that Splunk does not receive the same violations next time it
		// queries with the NewCheckpoint value.
		// TODO(ROX-6689): drop sorting and just find max ViolationTime while iterating through violations.
		response.NewCheckpoint = response.Violations[len(response.Violations)-1].GetViolationInfo().GetViolationTime()
	}

	return &response, nil
}

func getCheckpointValue(r *http.Request) (*types.Timestamp, time.Time, error) {
	param := r.URL.Query().Get(checkpointQueryParam)
	if param == "" {
		return fallbackCheckpoint, fallbackCheckpointTime, nil
	}
	ts, t, err := parseTimestamp(param)
	if err != nil {
		return nil, time.Time{}, httputil.Errorf(http.StatusBadRequest,
			"could not parse query parameter %q value %q as timestamp (try the following format: %s=%s): %s",
			checkpointQueryParam, param, checkpointQueryParam, fallbackCheckpointValue, err)
	}
	return ts, t, nil
}

func queryAlerts(ctx context.Context, alertDS datastore.DataStore, fromTime time.Time) ([]*storage.Alert, error) {
	// Alert searcher limits results to only certain (open) alert states if state query wasn't explicitly provided.
	// See https://github.com/stackrox/rox/blob/fe0447b512623111b78f5f7f1eb22f39e3e70cb3/central/alert/datastore/internal/search/searcher_impl.go#L142
	// This isn't desirable for Splunk integration because we want Splunk to know about all alerts irrespective of their
	// current state in StackRox.
	// The following explicitly instructs the search to return alerts in _all_ states.
	query := search.NewQueryBuilder().AddStrings(search.ViolationState, "*")

	// Here we add filtering to only receive Alerts that were updated after the timestamp provided as the checkpoint.
	// This is an optimization that allows to reduce the amount of data read from the datastore.
	// Alert times are updated each time Alert receives a new violation and so by applying this filtering we are picking
	// up both brand new alerts and old alerts that had new violations. Further on we still filter violations according
	// to violation timestamp to make sure that violations of some alert with a history are not included if they're
	// before the checkpoint but violations after the checkpoint are.
	// The downstream timestamp querying supports granularity up to a second and a peculiar timestamp format therefore
	// we leverage what it provides.
	// See https://github.com/stackrox/rox/blob/master/pkg/search/blevesearch/time_query.go
	query = query.AddStrings(search.ViolationTime, ">="+fromTime.Format("01/02/2006 3:04:05 PM MST"))

	return alertDS.SearchRawAlerts(ctx, query.ProtoQuery())
}

func extractViolations(alert *storage.Alert, fromTimestamp *types.Timestamp) ([]*integrations.SplunkViolation, error) {
	var result []*integrations.SplunkViolation
	seenViolations := false

	// TODO: reuse deploymentInfo, policyInfo for efficiency. There's no need to parse the same data for every violation.

	if processViolation := alert.GetProcessViolation(); processViolation != nil {
		if len(processViolation.GetProcesses()) == 0 {
			log.Warnw("Detected ProcessViolation without ProcessIndicators. No process violations can be extracted from this Alert.", zap.String("Alert.Id", alert.GetId()))
		}
		for _, procIndicator := range processViolation.GetProcesses() {
			seenViolations = true

			timestamp := getProcessViolationTime(alert, procIndicator)
			if timestamp.Compare(fromTimestamp) <= 0 {
				continue
			}

			violationInfo := extractProcessViolationInfo(alert, processViolation, procIndicator)
			result = append(result, &integrations.SplunkViolation{
				ViolationInfo:  violationInfo,
				AlertInfo:      extractAlertInfo(alert, violationInfo),
				ProcessInfo:    extractProcessInfo(alert.GetId(), procIndicator),
				DeploymentInfo: extractDeploymentInfo(alert, procIndicator),
				PolicyInfo:     extractPolicyInfo(alert.GetId(), alert.GetPolicy()),
			})
		}
	}
	for _, v := range alert.GetViolations() {
		seenViolations = true

		timestamp := getNonProcessViolationTime(alert, v)
		if timestamp.Compare(fromTimestamp) <= 0 {
			continue
		}

		violationInfo, err := extractNonProcessViolationInfo(alert, v)
		if err != nil {
			return nil, err
		}
		result = append(result, &integrations.SplunkViolation{
			ViolationInfo:  violationInfo,
			AlertInfo:      extractAlertInfo(alert, violationInfo),
			DeploymentInfo: extractDeploymentInfo(alert, nil),
			PolicyInfo:     extractPolicyInfo(alert.GetId(), alert.GetPolicy()),
		})
	}

	if !seenViolations {
		log.Warnw("Did not detect any extractable violations from the Alert. Information about the alert will not be available in Splunk.", zap.String("Alert.Id", alert.GetId()))
	}

	return result, nil
}

// generateViolationID creates a string from alertID plus hash of storage.Alert_Violation contents for use as violationId.
// This is because storage.Alert_Violation does not have own Id.
// We're using full content of storage.Alert_Violation and not merely storage.Alert_Violation.Time for ID generation
// because non-runtime alerts don't have Time and for other types Time is not guaranteed to be unique (e.g. time of
// ingestion for multiple events).
// It still may happen that different violations in storage.Alert have the same content and we'll produce the same Ids
// for them. Let's see if this ever becomes a problem.
func generateViolationID(alertID string, v *storage.Alert_Violation) (string, error) {
	data, err := v.Marshal()
	if err != nil {
		return "", err
	}
	alertUUID := uuid.FromStringOrNil(alertID)
	// This works around the case when alertID is just some string and not a UUID.
	// In the end we just need IDs with uniqueness so this should be good enough.
	alertUUID = uuid.NewV5(alertUUID, alertID)
	return uuid.NewV5(alertUUID, hex.Dump(data)).String(), nil
}

func getProcessViolationTime(fromAlert *storage.Alert, fromProcIndicator *storage.ProcessIndicator) *types.Timestamp {
	timestamp := fromProcIndicator.GetSignal().GetTime()
	if timestamp == nil {
		// As a fallback when process violation does not have own time on the record take Alert's last seen time to
		// provide at least some value.
		timestamp = fromAlert.GetTime()
	}
	return timestamp
}

func extractProcessViolationInfo(fromAlert *storage.Alert, fromProcViolation *storage.Alert_ProcessViolation, fromProcIndicator *storage.ProcessIndicator) *integrations.SplunkViolation_ViolationInfo {
	// fromProcViolation.Message can change over time. For example, first it begins like this
	//   Binary '/usr/bin/nmap' executed with arguments '--help' under user ID 0
	// On the next violation, the message changes to
	//   Binary '/usr/bin/nmap' executed with 2 different arguments under user ID 0
	// and so on. Here's another example
	//   Binaries '/usr/bin/apt' and '/usr/bin/dpkg' executed with 5 different arguments under 2 different user IDs
	// We still map it as best effort.
	return &integrations.SplunkViolation_ViolationInfo{
		ViolationId:        fromProcIndicator.GetId(),
		ViolationMessage:   fromProcViolation.GetMessage(),
		ViolationType:      integrations.SplunkViolation_ViolationInfo_PROCESS_EVENT,
		ViolationTime:      getProcessViolationTime(fromAlert, fromProcIndicator),
		PodId:              fromProcIndicator.GetPodId(),
		PodUid:             fromProcIndicator.GetPodUid(),
		ContainerName:      fromProcIndicator.GetContainerName(),
		ContainerStartTime: fromProcIndicator.GetContainerStartTime(),
		ContainerId:        fromProcIndicator.GetSignal().GetContainerId(),
	}
}

func getNonProcessViolationTime(fromAlert *storage.Alert, fromViolation *storage.Alert_Violation) *types.Timestamp {
	timestamp := fromViolation.GetTime()
	if timestamp == nil {
		// Generic violations don't have own timestamp therefore we report them with Alert's last seen time.
		// This way all generic violations under the same Alert will have the same most recent violation time even
		// though they might have happened at different moments.
		// This means we'll likely resurrect old already seen violations when the alert gets updated, i.e. over-alarm.
		// Perhaps that's better than set ViolationTime to fromAlert.FirstOccurred which will under-alarm.
		// TODO(ROX-6706): simplify this after timestamps are added to all violations
		timestamp = fromAlert.GetTime()
	}
	return timestamp
}

func extractNonProcessViolationInfo(fromAlert *storage.Alert, fromViolation *storage.Alert_Violation) (*integrations.SplunkViolation_ViolationInfo, error) {
	id, err := generateViolationID(fromAlert.GetId(), fromViolation)
	if err != nil {
		return nil, err
	}

	var msgAttrs []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr
	if kvs := fromViolation.GetKeyValueAttrs(); kvs != nil {
		msgAttrs = kvs.Clone().GetAttrs()
	}

	var podID, containerName string
	for _, kv := range msgAttrs {
		if kv.Key == printer.PodKey {
			podID = kv.Value
		}
		if kv.Key == printer.ContainerKey {
			containerName = kv.Value
		}
	}

	typ := integrations.SplunkViolation_ViolationInfo_UNKNOWN
	switch fromViolation.GetType() {
	case storage.Alert_Violation_GENERIC:
		typ = integrations.SplunkViolation_ViolationInfo_GENERIC
	case storage.Alert_Violation_K8S_EVENT:
		typ = integrations.SplunkViolation_ViolationInfo_K8S_EVENT
	case storage.Alert_Violation_NETWORK_FLOW:
		typ = integrations.SplunkViolation_ViolationInfo_NETWORK_FLOW
	}

	return &integrations.SplunkViolation_ViolationInfo{
		ViolationId:                id,
		ViolationMessage:           fromViolation.GetMessage(),
		ViolationMessageAttributes: msgAttrs,
		ViolationType:              typ,
		ViolationTime:              getNonProcessViolationTime(fromAlert, fromViolation),
		PodId:                      podID,
		ContainerName:              containerName,
	}, nil
}

func extractProcessInfo(alertID string, from *storage.ProcessIndicator) *integrations.SplunkViolation_ProcessInfo {
	var signal storage.ProcessSignal
	var pid, uid, gid *types.UInt32Value
	var lineage []*storage.ProcessSignal_LineageInfo

	if from.GetSignal() != nil {
		signal = *from.GetSignal()
		pid = &types.UInt32Value{Value: signal.GetPid()}
		uid = &types.UInt32Value{Value: signal.GetUid()}
		gid = &types.UInt32Value{Value: signal.GetGid()}

		lineage = make([]*storage.ProcessSignal_LineageInfo, 0, len(signal.GetLineageInfo()))
		for _, x := range signal.GetLineageInfo() {
			lineage = append(lineage, x.Clone())
		}
	} else {
		log.Warnw("Detected ProcessIndicator without inner ProcessSignal. Resulting process details will be incomplete.", zap.String("ProcessIndicator.Id", from.GetId()), zap.String("Alert.Id", alertID))
	}

	return &integrations.SplunkViolation_ProcessInfo{
		ProcessViolationId:  from.GetId(),
		ProcessSignalId:     signal.GetId(),
		ProcessCreationTime: signal.GetTime(),
		ProcessName:         signal.GetName(),
		ProcessArgs:         signal.GetArgs(),
		ExecFilePath:        signal.GetExecFilePath(),
		Pid:                 pid,
		ProcessUid:          uid,
		ProcessGid:          gid,
		ProcessLineageInfo:  lineage,
	}
}

func extractAlertInfo(from *storage.Alert, violationInfo *integrations.SplunkViolation_ViolationInfo) *integrations.SplunkViolation_AlertInfo {
	var firstOccurred *types.Timestamp
	if violationInfo.GetViolationType() == integrations.SplunkViolation_ViolationInfo_GENERIC {
		// Generic violations don't have own timestamp and so we assign Alert's last seen time to violation time.
		// That is not accurate and therefore here we add Alert's first seen time as an additional information element.
		firstOccurred = from.GetFirstOccurred()
		// We're NOT doing the same for other violation types because the user can get confused thinking that Alert's
		// first occurred time applies to the _specific_ violation. I.e. external users don't necessarily know
		// relationship between Alerts anv Violations is 1:n in StackRox.
	}

	return &integrations.SplunkViolation_AlertInfo{
		AlertId:            from.GetId(),
		LifecycleStage:     from.GetLifecycleStage(),
		AlertTags:          from.GetTags(),
		AlertFirstOccurred: firstOccurred,
	}
	// from.State and from.SnoozeTill are ignored because they might change over time.
}

func extractPolicyInfo(alertID string, from *storage.Policy) *integrations.SplunkViolation_PolicyInfo {
	if from == nil {
		log.Warnw("Detected Alert without Policy. Resulting violation item will not have policy details.", zap.String("Alert.Id", alertID))
		return nil
	}

	lcStages := make([]string, 0, len(from.GetLifecycleStages()))
	for _, x := range from.GetLifecycleStages() {
		lcStages = append(lcStages, x.String())
	}

	return &integrations.SplunkViolation_PolicyInfo{
		PolicyId:              from.GetId(),
		PolicyName:            from.GetName(),
		PolicyDescription:     from.GetDescription(),
		PolicyRationale:       from.GetRationale(),
		PolicyCategories:      from.GetCategories(),
		PolicyLifecycleStages: lcStages,
		PolicySeverity:        from.GetSeverity().String(),
		PolicyVersion:         from.GetPolicyVersion(),
	}
	// from.PolicySections are not mapped because sufficient information is provided other policy fields and violation
	// message.
}

func extractDeploymentInfo(from *storage.Alert, fromProc *storage.ProcessIndicator) *integrations.SplunkViolation_DeploymentInfo {
	var res integrations.SplunkViolation_DeploymentInfo

	switch e := from.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		containers := make([]*storage.Alert_Deployment_Container, 0, len(e.Deployment.GetContainers()))
		for _, x := range e.Deployment.GetContainers() {
			containers = append(containers, x.Clone())
		}

		res = integrations.SplunkViolation_DeploymentInfo{
			DeploymentId:          e.Deployment.GetId(),
			DeploymentName:        e.Deployment.GetName(),
			DeploymentType:        e.Deployment.GetType(),
			DeploymentNamespace:   e.Deployment.GetNamespace(),
			DeploymentNamespaceId: e.Deployment.GetNamespaceId(),
			DeploymentLabels:      e.Deployment.GetLabels(),
			ClusterId:             e.Deployment.GetClusterId(),
			ClusterName:           e.Deployment.GetClusterName(),
			DeploymentContainers:  containers,
			DeploymentAnnotations: e.Deployment.GetAnnotations(),
		}
		// e.Deployment.Inactive not mapped because it might change
	case *storage.Alert_Image:
		res = integrations.SplunkViolation_DeploymentInfo{
			DeploymentImage: e.Image.Clone(),
		}
	default:
		log.Warnw("Alert.Entity unrecognized or not set. Resulting violation item will not have deployment details.", zap.String("Alert.Id", from.GetId()))
	}

	if fromProc != nil {
		// Process violations come with own information about deployment where the violation happened.
		// I decided to trust that information more, and, in unlikely case when there are discrepancies between info
		// coming from process and in Alert itself, we give priority to process info.

		resetIfDiffer := func(field, v1, v2 string) {
			if v1 != "" && v2 != "" && v1 != v2 {
				log.Warnw(
					fmt.Sprintf("Alert %s=%q does not correspond to %s=%q of recorded process violation.", field, v1, field, v2)+
						" Resulting deployment details will not be complete.",
					zap.String("Alert.Id", from.GetId()))
				res = integrations.SplunkViolation_DeploymentInfo{}
			}
		}
		resetIfDiffer("DeploymentId", res.GetDeploymentId(), fromProc.GetDeploymentId())
		resetIfDiffer("Namespace", res.GetDeploymentNamespace(), fromProc.GetNamespace())
		resetIfDiffer("ClusterId", res.GetClusterId(), fromProc.GetClusterId())

		if fromProc.GetDeploymentId() != "" {
			res.DeploymentId = fromProc.GetDeploymentId()
		}
		if fromProc.GetNamespace() != "" {
			res.DeploymentNamespace = fromProc.GetNamespace()
		}
		if fromProc.GetClusterId() != "" {
			res.ClusterId = fromProc.GetClusterId()
		}
	}

	return &res
}
