package models

import (
	"context"
	"time"
)

// AuthMode controls which ACS authentication scheme the importer uses.
type AuthMode string

const (
	AuthModeToken AuthMode = "token"
	AuthModeBasic AuthMode = "basic"
)

// Config holds all resolved configuration for a single importer run.
type Config struct {
	ACSEndpoint        string   // from --endpoint or ROX_ENDPOINT
	AuthMode           AuthMode // auto-inferred from env vars (ROX_API_TOKEN / ROX_ADMIN_PASSWORD)
	Username           string   // from --username or ROX_ADMIN_USER (default "admin")
	CONamespace        string   // empty when COAllNamespaces=true
	COAllNamespaces    bool
	ACSClusterID       string // auto-discovered per context; set at runtime during iteration
	DryRun             bool
	ReportJSON         string
	RequestTimeout     time.Duration
	MaxRetries         int
	CACertFile         string
	InsecureSkipVerify bool
	OverwriteExisting  bool
	Contexts           []string // opt-in --context filter; empty means all contexts
}

// Severity classifies how severe a Problem is.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Category classifies what kind of issue a Problem represents.
type Category string

const (
	CategoryInput      Category = "input"
	CategoryMapping    Category = "mapping"
	CategoryConflict   Category = "conflict"
	CategoryAuth       Category = "auth"
	CategoryAPI        Category = "api"
	CategoryRetry      Category = "retry"
	CategoryValidation Category = "validation"
)

// Problem is a structured issue entry recorded during an importer run.
type Problem struct {
	Severity    Severity `json:"severity"`
	Category    Category `json:"category"`
	ResourceRef string   `json:"resourceRef"` // "namespace/name" or synthetic
	Description string   `json:"description"`
	FixHint     string   `json:"fixHint"`
	Skipped     bool     `json:"skipped"`
}

// ACSSchedule is the schedule portion of an ACS scan configuration.
type ACSSchedule struct {
	IntervalType string          `json:"intervalType,omitempty"`
	Hour         int32           `json:"hour"`
	Minute       int32           `json:"minute"`
	Weekly       *ACSWeekly      `json:"weekly,omitempty"`
	DaysOfWeek   *ACSDaysOfWeek  `json:"daysOfWeek,omitempty"`
	DaysOfMonth  *ACSDaysOfMonth `json:"daysOfMonth,omitempty"`
}

// ACSWeekly holds the day-of-week for a weekly ACS schedule.
type ACSWeekly struct {
	Day int32 `json:"day"`
}

// ACSDaysOfWeek holds multiple days for a multi-day-of-week ACS schedule.
type ACSDaysOfWeek struct {
	Days []int32 `json:"days"`
}

// ACSDaysOfMonth holds days for a monthly ACS schedule.
type ACSDaysOfMonth struct {
	Days []int32 `json:"days"`
}

// ACSBaseScanConfig is the scanConfig sub-object in an ACS create payload.
type ACSBaseScanConfig struct {
	OneTimeScan  bool         `json:"oneTimeScan"`
	Profiles     []string     `json:"profiles"`
	ScanSchedule *ACSSchedule `json:"scanSchedule,omitempty"`
	Description  string       `json:"description"`
}

// ACSCreatePayload is the request body for POST /v2/compliance/scan/configurations
// and PUT /v2/compliance/scan/configurations/{id}.
type ACSCreatePayload struct {
	ScanName   string            `json:"scanName"`
	ScanConfig ACSBaseScanConfig `json:"scanConfig"`
	Clusters   []string          `json:"clusters"`
}

// ACSConfigSummary is a single entry from the ACS list response.
type ACSConfigSummary struct {
	ID       string `json:"id"`
	ScanName string `json:"scanName"`
}

// ACSListResponse matches the JSON from GET /v2/compliance/scan/configurations.
type ACSListResponse struct {
	Configurations []ACSConfigSummary `json:"configurations"`
	TotalCount     int32              `json:"totalCount"`
}

// ReportMeta is metadata written at the top of the JSON report.
type ReportMeta struct {
	Timestamp      string `json:"timestamp"`
	DryRun         bool   `json:"dryRun"`
	NamespaceScope string `json:"namespaceScope"`
	Mode           string `json:"mode"` // always "create-only"
}

// ReportCounts summarises action totals for the JSON report.
type ReportCounts struct {
	Discovered int `json:"discovered"`
	Create     int `json:"create"`
	Update     int `json:"update"`
	Skip       int `json:"skip"`
	Failed     int `json:"failed"`
}

// ReportItemSource identifies the CO source for one report item.
type ReportItemSource struct {
	Namespace       string `json:"namespace"`
	BindingName     string `json:"bindingName"`
	ScanSettingName string `json:"scanSettingName"`
}

// ReportItem records the outcome for one ScanSettingBinding.
type ReportItem struct {
	Source          ReportItemSource `json:"source"`
	Action          string           `json:"action"` // create|skip|fail
	Reason          string           `json:"reason"`
	Attempts        int              `json:"attempts"`
	ACSScanConfigID string           `json:"acsScanConfigId,omitempty"`
	Error           string           `json:"error,omitempty"`
}

// Report is the top-level structure written to --report-json.
type Report struct {
	Meta     ReportMeta   `json:"meta"`
	Counts   ReportCounts `json:"counts"`
	Items    []ReportItem `json:"items"`
	Problems []Problem    `json:"problems"`
}

// ACSClusterInfo represents a cluster managed by ACS.
type ACSClusterInfo struct {
	ID                string // ACS cluster UUID
	Name              string // cluster display name
	ProviderClusterID string // from status.providerMetadata.cluster.id (e.g. OpenShift cluster ID)
}

// ACSClient is the interface for ACS API operations.
type ACSClient interface {
	Preflight(ctx context.Context) error
	ListScanConfigurations(ctx context.Context) ([]ACSConfigSummary, error)
	CreateScanConfiguration(ctx context.Context, payload ACSCreatePayload) (string, error)
	UpdateScanConfiguration(ctx context.Context, id string, payload ACSCreatePayload) error
	ListClusters(ctx context.Context) ([]ACSClusterInfo, error)
}
