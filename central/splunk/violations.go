package splunk

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/generated/api/integrations"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	checkpointQueryParam = "from_checkpoint"
	fallbackCheckpoint   = "2000-01-01T00:00:00.000Z"

	// A violation gets the current timestamp after it is detected. However, it takes some time before the violation is
	// stored in the database because it needs to go through enrichment and get to central. For example, the violation
	// occurs now at 8:09:01 (p.m.) but is saved in the database at 8:09:06, i.e. with 5 seconds delay. If we try to
	// query for this violation at 8:09:04, we will not find it even though it occurred before that moment. It will not
	// be visible until after 8:09:06.
	// We introduce a buffer duration - eventualConsistencyMargin - which is subtracted from time.Now() when
	// querying by Alert timestamp and when filtering violations in order not to lose recent violations that were
	// detected but not yet seen in the database. Ten seconds were chosen based on our understanding how quickly
	// violations are persisted.
	eventualConsistencyMargin = 10 * time.Second
)

var (
	log = logging.LoggerForModule()
)

type paginationSettings struct {
	// limit number of alerts returned from database
	maxAlertsFromQuery int32
	// approximately this many violations we want in the response
	violationsPerResponse int
}

// defaultPaginationSettings provide pagination limits used in production.
//
// Average JSON SplunkViolation size is around 2 kilobytes. 5000 violations per response were chosen so that 10Mb
// response fits well under ~250Mb response sizes that I observed started failing during Splunk local imports.
// Splunk timeout appears to be between 1 and 2 minutes (seems slightly more than 1 minute).
//
// 500 alerts from query limit was chosen based on intuition that each Runtime Alert has on average 5-10 violations.
// In the worst case, Runtime Alert can have up to 40 violations (hard limit), and so the database query will return
// 4x violations than will be included in the response. I.e. the number 500 alerts can provide between 1/10 and 4x of
// desired number of violations.
// Note that non-runtime Alert currently results in a single SplunkViolation irrespective of how many
// storage.Violation-s are inside.
//
// More info and raw numbers in https://stack-rox.atlassian.net/browse/ROX-6868
var defaultPaginationSettings = paginationSettings{
	maxAlertsFromQuery:    500,
	violationsPerResponse: 5000,
}

var (
	// Set of keys to remove from the violationMessageAttributes field of a Kubernetes Event violation
	// This is done for so that we can reduce the amount of unnecessary bytes sent to Splunk for fields that can be inferred
	// via other fields, as Splunk charges by the data ingested.
	violationMessagesToRemoveForK8SEvent = set.NewFrozenStringSet(printer.ResourceURIKey)
)

// NewViolationsHandler provides violations data to Splunk on HTTP requests.
func NewViolationsHandler(alertDS datastore.DataStore) http.HandlerFunc {
	return newViolationsHandler(alertDS, defaultPaginationSettings)
}

// newViolationsHandler allows overriding paginationSettings during tests.
func newViolationsHandler(alertDS datastore.DataStore, pagination paginationSettings) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		res, err := getViolationsResponse(alertDS, r, pagination)
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

func getViolationsResponse(alertDS datastore.DataStore, r *http.Request, pagination paginationSettings) (*integrations.SplunkViolationsResponse, error) {
	checkpoint, err := getCheckpointValue(r)
	if err != nil {
		return nil, err
	}
	fromTimestamp, err := types.TimestampProto(checkpoint.fromTimestamp)
	if err != nil {
		return nil, err
	}
	toTimestamp, err := types.TimestampProto(checkpoint.toTimestamp)
	if err != nil {
		return nil, err
	}

	response := integrations.SplunkViolationsResponse{}

	for len(response.Violations) < pagination.violationsPerResponse {
		alerts, err := queryAlerts(r.Context(), alertDS, checkpoint, pagination.maxAlertsFromQuery)
		if err != nil {
			return nil, err
		}

		lastAlertID := ""

		for _, alert := range alerts {
			violations, err := extractViolations(alert, fromTimestamp, toTimestamp)
			if err != nil {
				return nil, err
			}
			response.Violations = append(response.Violations, violations...)

			if alert.GetId() > lastAlertID {
				lastAlertID = alert.GetId()
			}

			// We cannot strictly limit number of violations in the response without further complicating the checkpoint
			// format. Therefore we stop after we got enough violations and processed the entire Alert.
			if len(response.Violations) >= pagination.violationsPerResponse {
				break
			}
		}

		if lastAlertID != "" {
			// The next query should continue from the Alert following the last one where the processing stopped this time.
			checkpoint.fromAlertID = lastAlertID
		} else {
			// Advance the checkpoint timestamps if no more alerts were returned (for the current checkpoint).
			checkpoint = checkpoint.makeNextCheckpoint()
			break
		}
	}

	response.NewCheckpoint = checkpoint.String()

	return &response, nil
}

func getCheckpointValue(r *http.Request) (splunkCheckpoint, error) {
	param := r.URL.Query().Get(checkpointQueryParam)
	cp, err := parseCheckpointParam(param)
	if err != nil {
		return splunkCheckpoint{}, httputil.Errorf(http.StatusBadRequest,
			"error parsing or validating checkpoint parameter %q value %q (try making request without this query parameter to see the example format): %s",
			checkpointQueryParam, param, err)
	}
	return cp, nil
}

func queryAlerts(ctx context.Context, alertDS datastore.DataStore, checkpoint splunkCheckpoint, maxAlertsFromQuery int32) ([]*storage.Alert, error) {
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
	query = query.AddStrings(search.ViolationTime, ">="+checkpoint.fromTimestamp.Format("01/02/2006 3:04:05 PM MST"))

	pq := query.ProtoQuery()

	pq.Pagination = &apiV1.QueryPagination{
		Limit:  maxAlertsFromQuery,
		Offset: 0,
		SortOptions: []*apiV1.QuerySortOption{{
			Field:          search.DocID.String(),
			Reversed:       false,
			SearchAfterOpt: &apiV1.QuerySortOption_SearchAfter{SearchAfter: checkpoint.fromAlertID},
		}},
	}

	return alertDS.SearchRawAlerts(ctx, pq)
}

func extractViolations(alert *storage.Alert, fromTimestamp *types.Timestamp, toTimestamp *types.Timestamp) ([]*integrations.SplunkViolation, error) {
	var result []*integrations.SplunkViolation
	seenViolations := false

	policyInfo := extractPolicyInfo(alert.GetId(), alert.GetPolicy())
	deploymentInfo := extractDeploymentInfo(alert)

	if processViolation := alert.GetProcessViolation(); processViolation != nil {
		if len(processViolation.GetProcesses()) == 0 {
			log.Warnw("Detected ProcessViolation without ProcessIndicators. No process violations can be extracted from this Alert.", logging.AlertID(alert.GetId()))
		}
		for _, procIndicator := range processViolation.GetProcesses() {
			seenViolations = true

			timestamp := getProcessViolationTime(alert, procIndicator)
			if timestamp.Compare(fromTimestamp) <= 0 || timestamp.Compare(toTimestamp) > 0 {
				continue
			}

			violationInfo := extractProcessViolationInfo(alert, processViolation, procIndicator)
			result = append(result, &integrations.SplunkViolation{
				ViolationInfo: violationInfo,
				AlertInfo:     extractAlertInfo(alert, violationInfo),
				ProcessInfo:   extractProcessInfo(alert.GetId(), procIndicator),
				// Process alerts are on a deployment so we can make the assumption that has a DeploymentInfo
				EntityInfo: &integrations.SplunkViolation_DeploymentInfo_{
					DeploymentInfo: refineDeploymentInfo(alert.GetId(), deploymentInfo, procIndicator),
				},
				PolicyInfo: policyInfo,
			})
		}
	}
	var genericViolationMessage strings.Builder
	for _, v := range alert.GetViolations() {
		seenViolations = true

		timestamp := getNonProcessViolationTime(alert, v)
		if timestamp.Compare(fromTimestamp) <= 0 || timestamp.Compare(toTimestamp) > 0 {
			continue
		}

		if isGenericViolation(v) {
			if genericViolationMessage.Len() != 0 {
				genericViolationMessage.WriteString("\n")
			}
			genericViolationMessage.WriteString(v.GetMessage())
			// For generic violations we only collect messages, the actual violation record is created below after the loop.
			continue
		}

		violationInfo, err := extractNonProcessViolationInfo(alert, v)
		if err != nil {
			return nil, err
		}
		violation := integrations.SplunkViolation{
			ViolationInfo:   violationInfo,
			AlertInfo:       extractAlertInfo(alert, violationInfo),
			PolicyInfo:      policyInfo,
			NetworkFlowInfo: v.GetNetworkFlowInfo().Clone(),
		}

		addEntityInfoToSplunkViolation(alert, &violation, deploymentInfo)

		result = append(result, &violation)
	}
	if genericViolationMessage.Len() != 0 {
		violationInfo := extractGenericViolationInfo(alert, genericViolationMessage.String())
		violation := &integrations.SplunkViolation{
			ViolationInfo: violationInfo,
			AlertInfo:     extractAlertInfo(alert, violationInfo),
			PolicyInfo:    policyInfo,
		}
		addEntityInfoToSplunkViolation(alert, violation, deploymentInfo)
		result = append(result, violation)
	}

	if !seenViolations {
		log.Warnw("Did not detect any extractable violations from the Alert. Information about the alert will not be available in Splunk.", logging.AlertID(alert.GetId()))
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
		// Use alert timestamp as a fallback in case violation timestamp wasn't provided.
		timestamp = fromAlert.GetTime()
	}
	return timestamp
}

func extractNonProcessViolationInfo(fromAlert *storage.Alert, fromViolation *storage.Alert_Violation) (*integrations.SplunkViolation_ViolationInfo, error) {
	id, err := generateViolationID(fromAlert.GetId(), fromViolation)
	if err != nil {
		return nil, err
	}

	msgAttrs := extractViolationMessageAttrs(fromViolation)

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

func extractViolationMessageAttrs(fromViolation *storage.Alert_Violation) []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr {
	var msgAttrs []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr
	if kvs := fromViolation.GetKeyValueAttrs(); kvs != nil {
		msgAttrs = kvs.Clone().GetAttrs()
	}

	// Filter out some message attributes, but only for K8S Events
	// This is done so that we can reduce the amount of unnecessary bytes to Splunk for fields that can be inferred.
	if fromViolation.Type == storage.Alert_Violation_K8S_EVENT {
		var filteredAttrs []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr
		for _, kvp := range msgAttrs {
			if !violationMessagesToRemoveForK8SEvent.Contains(kvp.Key) {
				filteredAttrs = append(filteredAttrs, kvp)
			}
		}
		msgAttrs = filteredAttrs
	}

	return msgAttrs
}

func extractGenericViolationInfo(fromAlert *storage.Alert, message string) *integrations.SplunkViolation_ViolationInfo {
	// Generic (non-runtime) violations are squashed together and presented as one violation for Splunk.
	// Primarily that's because they don't have own timestamps. If they had timestamps, we could process and filter them
	// the same way as K8S events and network violations.
	// Splunk users will see a new SplunkViolation with growing violation message and with the same ID as before when a
	// new generic Violation gets added to the existing Alert.
	// TODO(ROX-6706): un-merge generic violations after timestamps are added to all violations.
	return &integrations.SplunkViolation_ViolationInfo{
		ViolationId:      fromAlert.GetId(),
		ViolationMessage: message,
		ViolationTime:    fromAlert.GetTime(),
		ViolationType:    integrations.SplunkViolation_ViolationInfo_GENERIC,
	}
}

// isGenericViolation checks if the violation doesn't have anything except of message.
// Such violations are non-runtimes and can be squashed together in one integrations.SplunkViolation when storage.Alert
// has many of them.
func isGenericViolation(violation *storage.Alert_Violation) bool {
	return violation.GetType() == storage.Alert_Violation_GENERIC && violation.GetTime() == nil && violation.GetMessageAttributes() == nil
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
		log.Warnw("Detected ProcessIndicator without inner ProcessSignal. Resulting process details will be incomplete.",
			logging.String("ProcessIndicator.Id", from.GetId()), logging.AlertID(alertID))
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
		AlertFirstOccurred: firstOccurred,
	}
	// from.State and from.SnoozeTill are ignored because they might change over time.
}

func extractPolicyInfo(alertID string, from *storage.Policy) *integrations.SplunkViolation_PolicyInfo {
	if from == nil {
		log.Warnw("Detected Alert without Policy. Resulting violation item will not have policy details.",
			logging.AlertID(alertID))
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

func extractDeploymentInfo(from *storage.Alert) *integrations.SplunkViolation_DeploymentInfo {
	var res integrations.SplunkViolation_DeploymentInfo

	switch e := from.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		containers := make([]*storage.Alert_Deployment_Container, 0, len(e.Deployment.GetContainers()))
		for _, x := range e.Deployment.GetContainers() {
			containers = append(containers, x.Clone())
		}

		// NOTE: For backwards compatibility with older TA deployments and existing Splunk queries, we still send DeploymentInfo.
		// Eventually we may want to migrate all data to ResourceInfo.
		// We are currently not sending duplicates in DeploymentInfo and ResourceInfo simultaneously because Splunk charges by data ingested and the duplicate would
		// increase charges for our customers. Instead we will ask them to read from either DeploymentInfo || ResourceInfo
		// using Splunk's coalesce function.
		// This is only being done for deployment and image alerts at the moment.
		// Resource-based alerts are new and receive data in ResourceInfo.
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
	case *storage.Alert_Resource_:
		// ignore for now. Resource cannot be converted to deployment. It will correctly get populated into its own entity later
		return nil
	default:
		log.Warnw("Alert.Entity unrecognized or not set. Resulting violation item will not have deployment details.", logging.AlertID(from.GetId()))
	}

	return &res
}

func extractResourceInfo(from *storage.Alert_Resource) *integrations.SplunkViolation_ResourceInfo {
	if from == nil {
		return nil
	}
	return &integrations.SplunkViolation_ResourceInfo{
		ResourceType: strings.Title(strings.ToLower(from.GetResourceType().String())), // capitalize it. Eg "Configmaps" instead of "CONFIGMAPS"
		Name:         from.GetName(),
		ClusterId:    from.GetClusterId(),
		ClusterName:  from.GetClusterName(),
		Namespace:    from.GetNamespace(),
	}
}

func addEntityInfoToSplunkViolation(from *storage.Alert, splunkViolation *integrations.SplunkViolation, deploymentInfo *integrations.SplunkViolation_DeploymentInfo) {
	if deploymentInfo != nil {
		splunkViolation.EntityInfo = &integrations.SplunkViolation_DeploymentInfo_{
			DeploymentInfo: deploymentInfo,
		}
	}

	// We know that deploymentInfo and resourceInfo can't both be non-nil at the same time so there's no risk of overwriting.
	if resource := from.GetResource(); resource != nil {
		splunkViolation.EntityInfo = &integrations.SplunkViolation_ResourceInfo_{
			ResourceInfo: extractResourceInfo(resource),
		}
	}
}

// refineDeploymentInfo must only be called for process violations.
// Process violations bring own information about deployment where the violation happened.
// I decided to trust that information more, and, in unlikely case when there are discrepancies between info
// coming from process and in Alert itself, we give priority to process info.
func refineDeploymentInfo(alertID string, deploymentInfo *integrations.SplunkViolation_DeploymentInfo, fromProc *storage.ProcessIndicator) *integrations.SplunkViolation_DeploymentInfo {
	if fromProc == nil {
		// If there's no process indicator for some weird reason, there's nothing to change in the deployment info and
		// we can pass it through so that the resulting record has at least the location of violation.
		return deploymentInfo
	}

	res := deploymentInfo.Clone()

	resetIfDiffer := func(field, v1, v2 string) {
		if v1 != "" && v2 != "" && v1 != v2 {
			log.Warnw(
				fmt.Sprintf("Alert %s=%q does not correspond to %s=%q of recorded process violation.", field, v1, field, v2)+
					" Resulting deployment details will not be complete.",
				logging.AlertID(alertID))
			res = &integrations.SplunkViolation_DeploymentInfo{}
		}
	}
	resetIfDiffer("DeploymentId", res.GetDeploymentId(), fromProc.GetDeploymentId())
	resetIfDiffer("Namespace", res.GetDeploymentNamespace(), fromProc.GetNamespace())
	resetIfDiffer("ClusterId", res.GetClusterId(), fromProc.GetClusterId())

	// If process indicator has some data and Alert's deployment does not, here we'll add it to the resulting struct.
	if fromProc.GetDeploymentId() != "" {
		res.DeploymentId = fromProc.GetDeploymentId()
	}
	if fromProc.GetNamespace() != "" {
		res.DeploymentNamespace = fromProc.GetNamespace()
	}
	if fromProc.GetClusterId() != "" {
		res.ClusterId = fromProc.GetClusterId()
	}

	return res
}

// splunkCheckpoint represents parsed checkpoint. Possible checkpoint formats are:
//  1. "FromTimestamp__ToTimestamp__FromAlertID"
//  2. "FromTimestamp"
//
// #2 is used as initial checkpoint setting in Splunk TA config and when there were no more alerts in the previous request.
// #1 is used when there were more alerts between FromTimestamp and ToTimestamp and response had to stop because
// sufficient number of violations was returned.
type splunkCheckpoint struct {
	fromTimestamp, toTimestamp time.Time
	fromAlertID                string
}

// parseCheckpointParam parses checkpoint value passed to API endpoint in query parameters.
// It also validates and prepares the checkpoint so that it is directly usable in the pagination logic.
// Probably this function is overly complex for what it should do but at least it isolates the rest of the code from
// the necessary "hackery".
func parseCheckpointParam(value string) (splunkCheckpoint, error) {
	adjustedNow := time.Now().UTC().Add(-eventualConsistencyMargin)

	if value == "" {
		// In case no checkpoint value was provided, we take the default value to make API return all data from the
		// beginning of time.
		value = fallbackCheckpoint
	}

	parts := strings.Split(value, "__")

	if len(parts) > 3 {
		return splunkCheckpoint{}, errors.Errorf("too many parts in checkpoint value %s: found %d expecting up to 3 parts", value, len(parts))
	}

	fromTs, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return splunkCheckpoint{}, errors.Wrap(err, "could not parse FromTimestamp")
	}
	if fromTs.After(adjustedNow) {
		// If API accepts a timestamp in the future, it will return incomplete data when this time moves into past
		// within eventual consistency margin from now. Therefore future timestamps are not allowed.
		// Note that Splunk will keep retrying requests with the same checkpoint until it starts getting 200 response
		// (and a different newCheckpoint value). Therefore, it is "ok" for Splunk users to set checkpoint in the
		// future. They won't see the data until that checkpoint timestamp.
		return splunkCheckpoint{}, errors.New("FromTimestamp must not be in the future or within eventual consistency margin")
	}

	var toTs time.Time
	if len(parts) > 1 {
		toTs, err = time.Parse(time.RFC3339Nano, parts[1])
		if err != nil {
			return splunkCheckpoint{}, errors.Wrap(err, "could not parse ToTimestamp")
		}
	} else {
		// If checkpoint consists from only FromTimestamp, i.e. no ToTimestamp and FromAlertID, ToTimestamp will become
		// the current instant (minus eventual consistency buffer) which allows to get all already persisted violations
		// between provided FromTimestamp and the current moment when the request is processed.
		toTs = adjustedNow
	}
	if toTs.After(adjustedNow) { // Same reasoning as for fromTs above.
		return splunkCheckpoint{}, errors.New("ToTimestamp must not be in the future or within eventual consistency margin")
	}

	if fromTs.After(toTs) {
		// This should not happen but we error out for the case we or users mess it up somehow.
		return splunkCheckpoint{}, errors.New("FromTimestamp must not be after ToTimestamp")
	}

	// If FromAlertID part isn't specified, we take empty string which should always be the lowest value in
	// lexicographical ordering of Alert IDs. This way all Alerts will be considered in processing of the request with
	// such checkpoint.
	fromAlertID := ""
	if len(parts) > 2 {
		fromAlertID = parts[2]
	}

	return splunkCheckpoint{
		fromTimestamp: fromTs,
		toTimestamp:   toTs,
		fromAlertID:   fromAlertID,
	}, nil
}

// makeNextCheckpoint creates a checkpoint that starts from the same instant where the given one ends.
// This function must be called only when no more Alerts were returned from the query for the given checkpoint.
func (c splunkCheckpoint) makeNextCheckpoint() splunkCheckpoint {
	return splunkCheckpoint{
		fromTimestamp: c.toTimestamp,
	}
}

// String formats checkpoint value to be parsable by parseCheckpointParam.
// Also, returned value should be given to Splunk via newCheckpoint.
// Note that String() might not necessarily create the same representation that was given to parseCheckpointParam(),
// and vice versa: parsing String() returned value may produce splunkCheckpoint struct with different field values.
// That's not an issue with String(), rather peculiarity of parseCheckpointParam implementation.
func (c splunkCheckpoint) String() string {
	if c.toTimestamp.IsZero() && c.fromAlertID == "" {
		// If both toTimestamp and fromAlertID are unset, they should not be present in the output, only fromTimestamp
		// must be included. E.g. "2021-03-30T09:58:00.1234Z".
		// This way our implementation can assign the _current_ instant to toTimestamp when Splunk requests again with
		// this checkpoint.
		return c.fromTimestamp.Format(time.RFC3339Nano)
	}
	return c.fromTimestamp.Format(time.RFC3339Nano) + "__" + c.toTimestamp.Format(time.RFC3339Nano) + "__" + c.fromAlertID
}
