package data

import (
	"bytes"
	"time"

	"github.com/golang/protobuf/jsonpb"
	licenseproto "github.com/stackrox/rox/generated/shared/license"
	"github.com/stackrox/rox/pkg/jsonutil"
	"google.golang.org/grpc/codes"
)

// GRPCInvocationStats contains telemetry data about GRPC API calls
type GRPCInvocationStats struct {
	Code  codes.Code `json:"code"`
	Count int64      `json:"count"`
}

// HTTPInvocationStats contains telemetry data about HTTP API calls
type HTTPInvocationStats struct {
	StatusCode int   `json:"statusCode"` // HTTP status code, or -1 if there was a write error.
	Count      int64 `json:"count"`
}

// PanicStats contains telemetry data about API panics
type PanicStats struct {
	PanicDesc string `json:"panicDesc"` // Code location of panic, if the handler panicked.
	Count     int64  `json:"count"`
}

// HTTPRoute represents a particular HTTP route with stats for normal invocations and invocations that panicked.
type HTTPRoute struct {
	Route             string                 `json:"route"`
	NormalInvocations []*HTTPInvocationStats `json:"normalInvocations,omitempty"`
	PanicInvocations  []*PanicStats          `json:"panicInvocations,omitempty"`
}

// GRPCMethod represents a particular GRPC method with stats for normal invocations and invocations that panicked.
type GRPCMethod struct {
	Method            string                 `json:"method"`
	NormalInvocations []*GRPCInvocationStats `json:"normalInvocations,omitempty"`
	PanicInvocations  []*PanicStats          `json:"panicInvocations,omitempty"`
}

// APIStats contains telemetry data about different kinds of API calls
type APIStats struct {
	HTTP []*HTTPRoute  `json:"http,omitempty"`
	GRPC []*GRPCMethod `json:"grpc,omitempty"`
}

// BucketStats contains telemetry data about a DB bucket
type BucketStats struct {
	Name        string `json:"name"`
	UsedBytes   int64  `json:"usedBytes"`
	Cardinality int    `json:"cardinality"`
}

// TableStats contains telemetry data about a DB table
type TableStats struct {
	Name      string `json:"name"`
	RowCount  int64  `json:"rowCount"`
	TableSize int64  `json:"tableSizeBytes"`
	IndexSize int64  `json:"indexSizeBytes"`
	ToastSize int64  `json:"toastSizeBytes"`
}

// DatabaseDetailsStats contains telemetry details about sizing of databases
type DatabaseDetailsStats struct {
	DatabaseName string `json:"databaseName"`
	DatabaseSize int64  `json:"databaseSizeBytes"`
}

// DatabaseStats contains telemetry data about a DB
type DatabaseStats struct {
	Type              string                  `json:"type"`
	Path              string                  `json:"path"`
	AvailableBytes    int64                   `json:"availableBytes,omitempty"`
	DatabaseAvailable bool                    `json:"databaseAvailable,omitempty"`
	UsedBytes         int64                   `json:"usedBytes"`
	Buckets           []*BucketStats          `json:"buckets,omitempty"`
	Tables            []*TableStats           `json:"tables,omitempty"`
	DatabaseDetails   []*DatabaseDetailsStats `json:"databaseDetails,omitempty"`
	Errors            []string                `json:"errors,omitempty"`
}

// StorageInfo contains telemetry data about available disk, storage type, and the available databases
type StorageInfo struct {
	DiskCapacityBytes int64            `json:"diskCapacityBytes"`
	DiskUsedBytes     int64            `json:"diskUsedBytes"`
	StorageType       string           `json:"storageType,omitempty"`
	Databases         []*DatabaseStats `json:"dbs,omitempty"`
	Errors            []string         `json:"errors,omitempty"`
}

// LicenseJSON type encapsulates the License type and adds Marshal/Unmarshal methods
type LicenseJSON licenseproto.License

// MarshalJSON marshals license data to bytes, following jsonpb rules.
func (l *LicenseJSON) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := (&jsonpb.Marshaler{}).Marshal(&buf, (*licenseproto.License)(l)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalJSON unmarshals license JSON bytes into a License object, following jsonpb rules.
func (l *LicenseJSON) UnmarshalJSON(data []byte) error {
	return jsonutil.JSONBytesToProto(data, (*licenseproto.License)(l))
}

// CentralInfo contains telemetry data specific to StackRox' Central deployment
type CentralInfo struct {
	*RoxComponentInfo

	ID               string     `json:"id,omitempty"`
	InstallationTime *time.Time `json:"installationTime,omitempty"`

	Storage            *StorageInfo      `json:"storage,omitempty"`
	APIStats           *APIStats         `json:"apiStats,omitempty"`
	Orchestrator       *OrchestratorInfo `json:"orchestrator,omitempty"`
	AutoUpgradeEnabled bool              `json:"autoUpgradeEnabled,omitempty"`

	Errors []string `json:"errors,omitempty"`
}
