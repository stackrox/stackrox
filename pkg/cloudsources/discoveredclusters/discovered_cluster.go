package discoveredclusters

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/uuid"
)

var rootNamespaceUUID = uuid.FromStringOrPanic("1a379e9f-1ea4-4399-b7e1-4cd8b1ced3e3")

// GenerateDiscoveredClusterID returns a UUID5 based on the discovered cluster fields.
// Returns an empty string if the argument is invalid.
func GenerateDiscoveredClusterID(discoveredCluster *DiscoveredCluster) string {
	if discoveredCluster == nil {
		return ""
	}

	id := discoveredCluster.GetID()
	sourceID := discoveredCluster.GetCloudSourceID()
	if stringutils.AtLeastOneEmpty(id, sourceID) {
		return ""
	}
	key := strings.Join([]string{
		id,
		sourceID,
	}, ",")
	return uuid.NewV5(rootNamespaceUUID, key).String()
}

// DiscoveredCluster contains a sub set of *storage.DiscoveredCluster.
//
// Fields managed by the datastore, such as the dedup ID and timestamps,
// are excluded.
type DiscoveredCluster struct {
	ID                string
	Name              string
	Type              storage.ClusterMetadata_Type
	ProviderType      storage.DiscoveredCluster_Metadata_ProviderType
	Region            string
	FirstDiscoveredAt *time.Time
	Status            storage.DiscoveredCluster_Status
	CloudSourceID     string
}

// GetID returns the discovered cluster ID.
func (d *DiscoveredCluster) GetID() string {
	if d != nil {
		return d.ID
	}
	return ""
}

// GetName returns the discovered cluster name.
func (d *DiscoveredCluster) GetName() string {
	if d != nil {
		return d.Name
	}
	return ""
}

// GetType returns the discovered cluster type.
func (d *DiscoveredCluster) GetType() storage.ClusterMetadata_Type {
	if d != nil {
		return d.Type
	}
	return storage.ClusterMetadata_UNSPECIFIED
}

// GetProviderType returns the discovered cluster provider type.
func (d *DiscoveredCluster) GetProviderType() storage.DiscoveredCluster_Metadata_ProviderType {
	if d != nil {
		return d.ProviderType
	}
	return storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_UNSPECIFIED
}

// GetRegion returns the discovered cluster region.
func (d *DiscoveredCluster) GetRegion() string {
	if d != nil {
		return d.Region
	}
	return ""
}

// GetFirstDiscoveredAt returns the first discovered at timestamp.
func (d *DiscoveredCluster) GetFirstDiscoveredAt() *time.Time {
	if d != nil {
		return d.FirstDiscoveredAt
	}
	return nil
}

// GetStatus returns the discovered cluster status.
func (d *DiscoveredCluster) GetStatus() storage.DiscoveredCluster_Status {
	if d != nil {
		return d.Status
	}
	return storage.DiscoveredCluster_STATUS_UNSPECIFIED
}

// GetCloudSourceID returns the discovered cluster cloud source ID.
func (d *DiscoveredCluster) GetCloudSourceID() string {
	if d != nil {
		return d.CloudSourceID
	}
	return ""
}

// Validate will validate the discovered cluster.
func (d *DiscoveredCluster) Validate() error {
	if d == nil {
		return errors.New("empty discovered cluster")
	}
	errorList := errorhelpers.NewErrorList("Validation")
	if d.GetID() == "" {
		errorList.AddString("discovered cluster ID must be defined")
	}
	if d.GetName() == "" {
		errorList.AddString("discovered cluster name must be defined")
	}
	if d.GetCloudSourceID() == "" {
		errorList.AddString("cloud source ID must be defined")
	}
	return errorList.ToError()
}
