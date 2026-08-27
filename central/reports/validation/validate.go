package validation

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	reportGen "github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator"
	snapshotDS "github.com/stackrox/rox/central/reports/snapshot/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	vulnRequestCommon "github.com/stackrox/rox/central/vulnmgmt/vulnerabilityrequest/common"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/uuid"
	k8sValidation "k8s.io/apimachinery/pkg/api/validate/content"
)

// Use this context only to
// 1) check if notifiers and collection attached to report config exist
// 2) Populating notifiers and collection in report snapshot
var allAccessCtx = sac.WithAllAccess(context.Background())

// Validator validates the requests to report service and generates job request for RunReport service
type Validator struct {
	reportConfigDatastore reportConfigDS.DataStore
	snapshotDatastore     snapshotDS.DataStore
	collectionDatastore   collectionDS.DataStore
	notifierDatastore     notifierDS.DataStore
}

// New Validator instance
func New(reportConfigDatastore reportConfigDS.DataStore, reportSnapshotDatastore snapshotDS.DataStore,
	collectionDatastore collectionDS.DataStore, notifierDatastore notifierDS.DataStore) *Validator {
	return &Validator{
		reportConfigDatastore: reportConfigDatastore,
		snapshotDatastore:     reportSnapshotDatastore,
		collectionDatastore:   collectionDatastore,
		notifierDatastore:     notifierDatastore,
	}
}

// ValidateReportConfiguration validates the given report configuration object
func (v *Validator) ValidateReportConfiguration(config *apiV2.ReportConfiguration) error {
	if config.GetName() == "" {
		return errox.InvalidArgs.New("Report configuration name is empty")
	}

	if err := v.validateSchedule(config); err != nil {
		return err
	}
	if err := v.validateNotifiers(config); err != nil {
		return err
	}
	if err := v.validateResourceScope(config); err != nil {
		return err
	}
	if err := v.validateReportFilters(config); err != nil {
		return err
	}

	return nil
}

func (v *Validator) validateSchedule(config *apiV2.ReportConfiguration) error {
	schedule := config.GetSchedule()
	if schedule == nil {
		return nil
	}
	switch schedule.GetIntervalType() {
	case apiV2.ReportSchedule_UNSET:
		return errox.InvalidArgs.New("report configuration schedule must be one of DAILY, WEEKLY, or MONTHLY")
	case apiV2.ReportSchedule_DAILY:
		if schedule.GetDaysOfWeek() != nil || schedule.GetDaysOfMonth() != nil {
			return errox.InvalidArgs.New("daily schedule must not specify days of week or days of month")
		}
	case apiV2.ReportSchedule_WEEKLY:
		if schedule.GetDaysOfWeek() == nil || len(schedule.GetDaysOfWeek().GetDays()) == 0 {
			return errox.InvalidArgs.New("report configuration must specify days of week for weekly schedule")
		}
		for _, day := range schedule.GetDaysOfWeek().GetDays() {
			if day < 0 || day > 6 {
				return errox.InvalidArgs.New("invalid schedule: days of the week can be Sunday (0) - Saturday(6)")
			}
		}
	case apiV2.ReportSchedule_MONTHLY:
		if schedule.GetDaysOfMonth() == nil || len(schedule.GetDaysOfMonth().GetDays()) == 0 {
			return errox.InvalidArgs.New("report configuration must specify days of the month for monthly schedule")
		}
		for _, day := range schedule.GetDaysOfMonth().GetDays() {
			if day != 1 && day != 15 {
				return errox.InvalidArgs.New("reports can be sent out only 1st or 15th day of the month")
			}
		}
	}
	return nil
}

func (v *Validator) validateNotifiers(config *apiV2.ReportConfiguration) error {
	notifiers := config.GetNotifiers()
	if len(notifiers) == 0 {
		if config.GetSchedule() != nil {
			return errox.InvalidArgs.New("report configurations with a schedule must specify a notifier")
		}
		return nil
	}
	for _, notifier := range notifiers {
		if notifier.GetEmailConfig() == nil {
			return errox.InvalidArgs.New("notifier must specify an email notifier configuration")
		}
		if err := v.validateEmailConfig(notifier.GetEmailConfig()); err != nil {
			return err
		}
	}
	return nil
}

func (v *Validator) validateEmailConfig(emailConfig *apiV2.EmailNotifierConfiguration) error {
	if emailConfig.GetNotifierId() == "" {
		return errox.InvalidArgs.New("report configuration must specify a valid email notifier")
	}
	if len(emailConfig.GetMailingLists()) == 0 {
		return errox.InvalidArgs.New("report configuration must specify at least one email recipient to send the report to")
	}
	subjectMaxLen := env.ReportCustomEmailSubjectMaxLen.IntegerSetting()
	if len(emailConfig.GetCustomSubject()) > subjectMaxLen {
		return errox.InvalidArgs.Newf("custom email subject must be fewer than %d characters", subjectMaxLen)
	}
	bodyMaxLen := env.ReportCustomEmailBodyMaxLen.IntegerSetting()
	if len(emailConfig.GetCustomBody()) > bodyMaxLen {
		return errox.InvalidArgs.Newf("custom email body must be fewer than %d characters", bodyMaxLen)
	}

	errorList := errorhelpers.NewErrorList("Invalid email addresses in mailing list: ")
	for _, addr := range emailConfig.GetMailingLists() {
		if _, err := mail.ParseAddress(addr); err != nil {
			errorList.AddError(errox.InvalidArgs.Newf("invalid email recipient address: %s", addr))
		}
	}
	if !errorList.Empty() {
		return errorList.ToError()
	}

	// Use allAccessCtx since report creator/updater might not have permissions for integrationSAC
	exists, err := v.notifierDatastore.Exists(allAccessCtx, emailConfig.GetNotifierId())
	if err != nil {
		return errors.Errorf("Error looking up attached notifier, Notifier ID: %s, Error: %s", emailConfig.GetNotifierId(), err)
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "Notifier with ID %s not found.", emailConfig.GetNotifierId())
	}
	return nil
}

func (v *Validator) validateResourceScope(config *apiV2.ReportConfiguration) error {
	scope := config.GetResourceScope()
	if scope == nil {
		return errox.InvalidArgs.New("report configuration must specify a valid resource scope")
	}

	// Nodes are cluster-level resources without namespace hierarchy
	if config.GetType() == apiV2.ReportConfiguration_NODE_VULNERABILITY {
		return v.validateNodeResourceScope(scope)
	}

	switch ref := scope.GetScopeReference().(type) {
	case *apiV2.ResourceScope_CollectionScope:
		return v.validateCollectionScope(ref.CollectionScope)
	case *apiV2.ResourceScope_EntityScope:
		if !features.VulnerabilityReportsEnhancedFiltering.Enabled() {
			return errox.InvalidArgs.New("report configuration must specify a valid collection as resource scope")
		}
		return validateEntityScope(ref.EntityScope)
	default:
		return errox.InvalidArgs.New("report configuration must specify a valid resource scope")
	}
}

func (v *Validator) validateNodeResourceScope(scope *apiV2.ResourceScope) error {
	entityScope, ok := scope.GetScopeReference().(*apiV2.ResourceScope_EntityScope)
	if !ok {
		return errox.InvalidArgs.New("node vulnerability reports must use entity scope (cluster-based scoping only)")
	}

	if entityScope.EntityScope == nil {
		return errox.InvalidArgs.New("entity scope cannot be nil")
	}

	if err := validateEntityScope(entityScope.EntityScope); err != nil {
		return err
	}

	// Nodes exist at cluster scope only - unlike deployments, they have no namespace association
	for _, rule := range entityScope.EntityScope.GetRules() {
		if rule.GetEntity() != apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER {
			return errox.InvalidArgs.Newf("node vulnerability reports only support cluster-level scoping, got: %s", rule.GetEntity())
		}
	}

	return nil
}

func (v *Validator) validateCollectionScope(collectionRef *apiV2.CollectionReference) error {
	if collectionRef == nil || collectionRef.GetCollectionId() == "" {
		return errox.InvalidArgs.New("report configuration must specify a valid collection ID")
	}
	collectionID := collectionRef.GetCollectionId()
	// Use allAccessCtx since report creator/updater might not have permissions for workflowAdministrationSAC
	exists, err := v.collectionDatastore.Exists(allAccessCtx, collectionID)
	if err != nil {
		return errors.Errorf("Error trying to lookup attached collection, Collection: %s, Error: %s", collectionID, err)
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "Collection %s not found.", collectionID)
	}
	return nil
}

// validateEntityScope validates the provided EntityScope and its rules.
// It returns an error if:
// 1. the scope is nil;
// 2. any rule has an unset entity or field;
// 3. a rule uses the unsupported (cluster, annotation) combination;
// 4. a duplicate (entity, field) pair appears
// 5. a rule has no values, or a label rule contains values that are not in `key=value` format.
func validateEntityScope(es *apiV2.EntityScope) error {
	if es == nil {
		return errox.InvalidArgs.New("report configuration must specify a valid resource scope: either a collection scope with a valid collection ID or a non-nil entity scope")
	}
	type entityFieldKey struct {
		entity apiV2.ScopeEntity
		field  apiV2.ScopeField
	}
	seen := set.NewSet[entityFieldKey]()
	for _, rule := range es.GetRules() {
		if rule.GetEntity() == apiV2.ScopeEntity_SCOPE_ENTITY_UNSET {
			return errox.InvalidArgs.Newf("unexpected entity scope rule: %s", rule.GetEntity())
		}
		if rule.GetField() == apiV2.ScopeField_FIELD_UNSET {
			return errox.InvalidArgs.Newf("unexpected entity in scope rule for %s with an unset field", rule.GetEntity())
		}
		// Cluster annotation is not indexed and therefore unsupported.
		if rule.GetEntity() == apiV2.ScopeEntity_SCOPE_ENTITY_CLUSTER && rule.GetField() == apiV2.ScopeField_FIELD_ANNOTATION {
			return errox.InvalidArgs.New("annotation field is not supported for cluster entity scope rules")
		}
		key := entityFieldKey{entity: rule.GetEntity(), field: rule.GetField()}
		if !seen.Add(key) {
			return errox.InvalidArgs.Newf(
				"one rule per (entity, field) pair in entity scope rules is expected; duplicate (entity, field) pair found: entity=%v field=%v", rule.GetEntity(), rule.GetField())
		}
		if len(rule.GetValues()) == 0 {
			return errox.InvalidArgs.Newf(
				"provide at least one matching value for entity=%v field=%v rule", rule.GetEntity(), rule.GetField())
		}
		isMapField := rule.GetField() == apiV2.ScopeField_FIELD_LABEL || rule.GetField() == apiV2.ScopeField_FIELD_ANNOTATION
		for _, rv := range rule.GetValues() {
			valOfValue := rv.GetValue()
			if isMapField {
				mapKey, mapValue, found := strings.Cut(valOfValue, "=")
				if !found {
					return errox.InvalidArgs.Newf("%v values must be in 'key=value' format", rule.GetField())
				}
				// Check the key for a Kubernetes qualified name.
				if rv.GetMatchType() == apiV2.MatchType_EXACT {
					if errs := k8sValidation.IsLabelKey(mapKey); len(errs) > 0 {
						return errox.InvalidArgs.Newf("invalid %v key %q: %s", rule.GetField(), mapKey, strings.Join(errs, "; "))
					}
				}
				valOfValue = mapValue
			}
			if rv.GetMatchType() == apiV2.MatchType_REGEX {
				if _, err := regexp.Compile(valOfValue); err != nil {
					return errox.InvalidArgs.CausedByf("invalid regex %q: %v", valOfValue, err)
				}
			}
		}
	}
	return nil
}

func (v *Validator) validateReportFilters(config *apiV2.ReportConfiguration) error {
	switch config.GetType() {
	case apiV2.ReportConfiguration_VULNERABILITY:
		return v.validateImageFilters(config.GetVulnReportFilters())
	case apiV2.ReportConfiguration_NODE_VULNERABILITY:
		return v.validateNodeFilters(config.GetNodeVulnReportFilters())
	default:
		return errox.InvalidArgs.New("unsupported report type")
	}
}

func (v *Validator) validateImageFilters(filters *apiV2.VulnerabilityReportFilters) error {
	if filters == nil {
		return errox.InvalidArgs.New("report configuration must include vulnerability report filters")
	}

	if len(filters.GetImageTypes()) == 0 {
		return errox.InvalidArgs.New("vulnerability report filters should specify which image types to scan for CVEs; " +
			"the valid options are 'DEPLOYED' and 'WATCHED'")
	}

	if filters.GetCvesSince() == nil {
		return errox.InvalidArgs.New("vulnerability report filters must specify how far back in time to look for CVEs; " +
			"the valid options are 'sinceLastSentScheduledReport', 'allVuln', and 'startDate'")
	}
	if features.VulnerabilityReportsEnhancedFiltering.Enabled() {
		if q := filters.GetQuery(); q != "" {
			if _, err := search.ParseQuery(q); err != nil {
				return errox.InvalidArgs.CausedByf("invalid query in vulnerability report filters: %v", err)
			}
		}
	}
	return nil
}

func (v *Validator) validateNodeFilters(filters *apiV2.NodeVulnerabilityReportFilters) error {
	if filters == nil {
		return errox.InvalidArgs.New("node vulnerability report filters cannot be nil")
	}

	// TODO: Add support for since_last_sent_scheduled_report and since_start_date filters
	// once FirstNodeOccurrence timestamp is available in the data model
	if filters.GetCvesSince() == nil {
		return errox.InvalidArgs.New("node vulnerability report filters must specify CVE time filter")
	}

	switch filters.GetCvesSince().(type) {
	case *apiV2.NodeVulnerabilityReportFilters_AllVuln:
		// Valid - this is the supported type
	default:
		return errox.InvalidArgs.New("only all vulnerabilities filter is currently supported for node reports (since_last_sent and since_start_date require FirstNodeOccurrence timestamp)")
	}

	if q := filters.GetQuery(); q != "" {
		if _, err := search.ParseQuery(q); err != nil {
			return errox.InvalidArgs.CausedByf("invalid query in node vulnerability report filters: %v", err)
		}
	}

	return nil
}

// ValidateAndGenerateReportRequest validates the report configuration for which report is requested and generates a report request
func (v *Validator) ValidateAndGenerateReportRequest(
	configID string,
	notificationMethod storage.ReportStatus_NotificationMethod,
	requestType storage.ReportStatus_RunMethod,
	requesterID authn.Identity,
) (*reportGen.ReportRequest, error) {
	config, found, err := v.reportConfigDatastore.GetReportConfiguration(allAccessCtx, configID)
	if err != nil {
		return nil, errors.Wrapf(err, "Error finding report configuration %s", configID)
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "Report configuration id not found %s", configID)
	}
	// Verify ResourceScope is non-nil
	if !common.HasValidResourceScope(config.GetResourceScope()) {
		return nil, errox.InvalidArgs.Newf(
			"report configuration '%s' has an empty resource scope (no collection ID or entity scope)",
			configID)
	}

	if notificationMethod == storage.ReportStatus_EMAIL && len(config.GetNotifiers()) == 0 {
		return nil, errox.InvalidArgs.New(
			"email request sent for a report configuration that does not have any email notifiers configured")
	}

	var collection *storage.ResourceCollection
	if collectionID := config.GetResourceScope().GetCollectionId(); collectionID != "" {
		var found bool
		var err error
		collection, found, err = v.collectionDatastore.Get(allAccessCtx, collectionID)
		if err != nil {
			return nil, errors.Wrapf(err, "Error finding collection ID '%s'", collectionID)
		}
		if !found {
			return nil, errors.Wrapf(errox.NotFound, "Collection ID '%s' not found", collectionID)
		}
	}

	notifierIDs := make([]string, 0, len(config.GetNotifiers()))
	for _, notifierConf := range config.GetNotifiers() {
		notifierIDs = append(notifierIDs, notifierConf.GetId())
	}
	protoNotifiers, err := v.notifierDatastore.GetManyNotifiers(allAccessCtx, notifierIDs)
	if err != nil {
		return nil, errors.Wrap(err, "Error finding attached notifiers")
	}
	if len(protoNotifiers) != len(notifierIDs) {
		return nil, errors.Wrap(errox.NotFound, "Some of the attached notifiers not found")
	}

	return &reportGen.ReportRequest{
		Collection:     collection,
		ReportSnapshot: generateReportSnapshot(config, collection, protoNotifiers, notificationMethod, requestType, requesterID),
	}, nil
}

// ValidateCancelReportRequest validates if the given requester can cancel the report job with job ID = reportID.
func (v *Validator) ValidateCancelReportRequest(reportID string, requester *storage.SlimUser) error {
	snapshot, found, err := v.snapshotDatastore.Get(allAccessCtx, reportID)
	if err != nil {
		return errors.Wrapf(err, "Error finding report snapshot with job ID '%s'.", reportID)
	}
	if !found {
		return errors.Wrapf(errox.NotFound, "Report snapshot with job ID '%s' does not exist", reportID)
	}

	switch snapshot.GetReportStatus().GetRunState() {
	case storage.ReportStatus_WAITING, storage.ReportStatus_PREPARING:
		// valid states for cancellation — fall through
	default:
		return errors.Wrapf(errox.InvalidArgs, "Cannot cancel. Report job ID '%s' has already completed execution.", reportID)
	}
	if requester.GetId() != snapshot.GetRequester().GetId() {
		return errors.Wrap(errox.NotAuthorized, "Report job cannot be cancelled by a user who did not request the report.")
	}
	return nil
}

// PersistReportSnapshot validates that the user does not already have a pending
// report for the same config, sets the snapshot to WAITING, and persists it to
// the database. Returns the report ID.
func (v *Validator) PersistReportSnapshot(ctx context.Context, snapshot *storage.ReportSnapshot) (string, error) {
	reportType := snapshot.GetType()
	if snapshot.GetVulnReportFilters() != nil && snapshot.GetReportStatus().GetReportRequestType() == storage.ReportStatus_ON_DEMAND {
		hasPending, err := v.doesUserHavePendingReport(snapshot.GetReportConfigurationId(), snapshot.GetRequester().GetId(), reportType)
		if err != nil {
			return "", err
		}
		if hasPending {
			return "", errors.Wrapf(errox.AlreadyExists, "User already has a report running for config ID '%s'",
				snapshot.GetReportConfigurationId())
		}
	}

	if snapshot.GetViewBasedVulnReportFilters() != nil {
		hasPending, err := v.doesUserHaveViewBasedPendingReport(snapshot.GetRequester().GetId(), reportType)
		if err != nil {
			return "", err
		}
		if hasPending {
			return "", errors.New("User already has a view based report queued")
		}
	}

	snapshot.ReportStatus.RunState = storage.ReportStatus_WAITING
	snapshot.ReportStatus.QueuedAt = protocompat.TimestampNow()
	reportID, err := v.snapshotDatastore.AddReportSnapshot(ctx, snapshot)
	if err != nil {
		return "", err
	}
	return reportID, nil
}

func (v *Validator) doesUserHavePendingReport(configID string, userID string, reportType storage.ReportSnapshot_ReportType) (bool, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportConfigID, configID).
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
		AddExactMatches(search.ReportRequestType, storage.ReportStatus_ON_DEMAND.String()).
		AddExactMatches(search.ReportType, reportType.String()).
		ProtoQuery()
	snapshots, err := v.snapshotDatastore.SearchReportSnapshots(allAccessCtx, query)
	if err != nil {
		return false, err
	}
	for _, snap := range snapshots {
		if snap.GetRequester().GetId() == userID {
			return true, nil
		}
	}
	return false, nil
}

func (v *Validator) doesUserHaveViewBasedPendingReport(userID string, reportType storage.ReportSnapshot_ReportType) (bool, error) {
	query := search.NewQueryBuilder().
		AddExactMatches(search.ReportState, storage.ReportStatus_WAITING.String(), storage.ReportStatus_PREPARING.String()).
		AddExactMatches(search.ReportRequestType, storage.ReportStatus_VIEW_BASED.String()).
		AddExactMatches(search.UserID, userID).
		AddExactMatches(search.ReportType, reportType.String()).
		ProtoQuery()
	count, err := v.snapshotDatastore.Count(allAccessCtx, query)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func collectionSnapshot(collection *storage.ResourceCollection) *storage.CollectionSnapshot {
	if collection == nil {
		return nil
	}
	return &storage.CollectionSnapshot{
		Id:   collection.GetId(),
		Name: collection.GetName(),
	}
}

func generateReportSnapshot(
	config *storage.ReportConfiguration,
	collection *storage.ResourceCollection,
	protoNotifiers []*storage.Notifier,
	notificationMethod storage.ReportStatus_NotificationMethod,
	requestType storage.ReportStatus_RunMethod,
	requesterID authn.Identity,
) *storage.ReportSnapshot {
	var requester *storage.SlimUser
	switch requestType {
	case storage.ReportStatus_ON_DEMAND:
		requester = &storage.SlimUser{
			Id:   requesterID.UID(),
			Name: stringutils.FirstNonEmpty(requesterID.FullName(), requesterID.FriendlyName()),
		}
	case storage.ReportStatus_SCHEDULED:
		requester = config.GetCreator()
	}

	var snapshotType storage.ReportSnapshot_ReportType
	switch config.GetType() {
	case storage.ReportConfiguration_VULNERABILITY:
		snapshotType = storage.ReportSnapshot_VULNERABILITY
	case storage.ReportConfiguration_NODE_VULNERABILITY:
		snapshotType = storage.ReportSnapshot_NODE_VULNERABILITY
	}

	snapshot := &storage.ReportSnapshot{
		ReportConfigurationId: config.GetId(),
		Name:                  config.GetName(),
		Description:           config.GetDescription(),
		Type:                  snapshotType,
		Collection:            collectionSnapshot(collection),
		Schedule:              config.GetSchedule(),
		Requester:             requester,
		ReportStatus: &storage.ReportStatus{
			RunState:                 storage.ReportStatus_WAITING,
			ReportRequestType:        requestType,
			ReportNotificationMethod: notificationMethod,
		},
	}

	if features.VulnerabilityReportsEnhancedFiltering.Enabled() || config.GetType() == storage.ReportConfiguration_NODE_VULNERABILITY {
		snapshot.ResourceScope = config.GetResourceScope().CloneVT()
	}

	switch config.GetType() {
	case storage.ReportConfiguration_VULNERABILITY:
		reportFilters := config.GetVulnReportFilters()
		if reportFilters != nil {
			reportFilters = reportFilters.CloneVT()
			if requestType == storage.ReportStatus_ON_DEMAND {
				reportFilters.AccessScopeRules = common.ExtractAccessScopeRules(requesterID)
			}
			snapshot.Filter = &storage.ReportSnapshot_VulnReportFilters{
				VulnReportFilters: reportFilters,
			}
		}
	case storage.ReportConfiguration_NODE_VULNERABILITY:
		nodeFilters := config.GetNodeVulnReportFilters()
		if nodeFilters != nil {
			nodeFilters = nodeFilters.CloneVT()
			if requestType == storage.ReportStatus_ON_DEMAND {
				nodeFilters.AccessScopeRules = common.ExtractAccessScopeRules(requesterID)
			}
			snapshot.Filter = &storage.ReportSnapshot_NodeVulnReportFilters{
				NodeVulnReportFilters: nodeFilters,
			}
		}
	}

	notifierSnaps := make([]*storage.NotifierSnapshot, 0, len(config.GetNotifiers()))

	for i, notifierConf := range config.GetNotifiers() {
		notifierSnaps = append(notifierSnaps, &storage.NotifierSnapshot{
			NotifierConfig: &storage.NotifierSnapshot_EmailConfig{
				EmailConfig: func() *storage.EmailNotifierConfiguration {
					cfg := notifierConf.GetEmailConfig()
					cfg.NotifierId = notifierConf.GetId()
					return cfg
				}(),
			},
			NotifierName: protoNotifiers[i].GetName(),
		})
	}
	snapshot.Notifiers = notifierSnaps
	return snapshot
}

// generateViewBasedRequestName generates request name for view based reports
func generateViewBasedRequestName(user *storage.SlimUser) string {
	shortName := getShortName(user)
	now := time.Now()
	date := now.Format("Jan02")
	year := now.Format("2006")
	shortUUID, _, _ := strings.Cut(uuid.NewV4().String(), "-")
	return fmt.Sprintf("%s-%s-%s-%s", shortName, strings.ToLower(date), year, shortUUID)
}

func getShortName(user *storage.SlimUser) string {
	if user == nil {
		return vulnRequestCommon.DefaultUserShortName
	}

	name := strings.ToUpper(user.GetName())
	parts := strings.Split(name, " ")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	firstName := stringutils.FirstNonEmpty(parts...)
	lastName := stringutils.LastNonEmpty(parts...)
	if firstName != "" && lastName != "" {
		return fmt.Sprintf("%c%c", firstName[0], lastName[0])
	}
	return vulnRequestCommon.DefaultUserShortName
}

// ValidateAndGenerateViewBasedReportRequest validates a view-based report request and constructs the scheduler payload.
func (v *Validator) ValidateAndGenerateViewBasedReportRequest(
	req *apiV2.ReportRequestViewBased,
	requesterID authn.Identity,
) (*reportGen.ReportRequest, error) {
	if req == nil {
		return nil, errox.InvalidArgs.New("empty request")
	}

	requester := &storage.SlimUser{
		Id:   requesterID.UID(),
		Name: stringutils.FirstNonEmpty(requesterID.FullName(), requesterID.FriendlyName()),
	}

	var snapshot *storage.ReportSnapshot

	switch req.GetType() {
	case apiV2.ReportRequestViewBased_VULNERABILITY:
		// Validate filters.
		vbFilters := req.GetViewBasedVulnReportFilters()
		if vbFilters == nil {
			return nil, errox.InvalidArgs.New("view-based vulnerability report filters must be provided")
		}

		// Convert API filters to storage filters.
		storageFilters := &storage.ViewBasedVulnerabilityReportFilters{
			Query:            vbFilters.GetQuery(),
			AccessScopeRules: common.ExtractAccessScopeRules(requesterID),
		}

		// Build report snapshot.
		snapshot = &storage.ReportSnapshot{
			Name:          generateViewBasedRequestName(requester),
			Type:          storage.ReportSnapshot_VULNERABILITY,
			AreaOfConcern: req.GetAreaOfConcern(),
			ReportStatus: &storage.ReportStatus{
				RunState:                 storage.ReportStatus_WAITING,
				ReportRequestType:        storage.ReportStatus_VIEW_BASED,
				ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
			},
			Filter: &storage.ReportSnapshot_ViewBasedVulnReportFilters{
				ViewBasedVulnReportFilters: storageFilters,
			},
			Requester: requester,
		}

	case apiV2.ReportRequestViewBased_NODE_VULNERABILITY:
		// Validate filters.
		nodeFilters := req.GetNodeVulnReportFilters()
		if nodeFilters == nil {
			return nil, errox.InvalidArgs.New("node vulnerability report filters must be provided")
		}

		if err := v.validateNodeFilters(nodeFilters); err != nil {
			return nil, err
		}

		// Convert API filters to storage filters.
		storageFilters := &storage.NodeVulnerabilityReportFilters{
			Query:            nodeFilters.GetQuery(),
			AccessScopeRules: common.ExtractAccessScopeRules(requesterID),
		}

		switch nodeFilters.GetCvesSince().(type) {
		case *apiV2.NodeVulnerabilityReportFilters_AllVuln:
			storageFilters.CvesSince = &storage.NodeVulnerabilityReportFilters_AllVuln{
				AllVuln: nodeFilters.GetAllVuln(),
			}
		default:
			return nil, errox.InvalidArgs.New("unsupported CVE time filter for node vulnerability reports")
		}

		// Build report snapshot.
		snapshot = &storage.ReportSnapshot{
			Name:          generateViewBasedRequestName(requester),
			Type:          storage.ReportSnapshot_NODE_VULNERABILITY,
			AreaOfConcern: req.GetAreaOfConcern(),
			ReportStatus: &storage.ReportStatus{
				RunState:                 storage.ReportStatus_WAITING,
				ReportRequestType:        storage.ReportStatus_VIEW_BASED,
				ReportNotificationMethod: storage.ReportStatus_DOWNLOAD,
			},
			Filter: &storage.ReportSnapshot_NodeVulnReportFilters{
				NodeVulnReportFilters: storageFilters,
			},
			Requester: requester,
		}

	default:
		return nil, errox.InvalidArgs.New("unsupported report type")
	}

	return &reportGen.ReportRequest{
		ReportSnapshot: snapshot,
	}, nil
}
