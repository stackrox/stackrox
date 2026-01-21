package datastore

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
)

// listSecretResponse is a helper struct for scanning ListSecret query results.
// The search framework will automatically use array_agg for the Types field
// since it comes from the secrets_files child table and is marked with ChildTableAgg flag.
type listSecretResponse struct {
	ID          string               `db:"secret_id"`
	Name        string               `db:"secret"`
	ClusterID   string               `db:"cluster_id"`
	ClusterName string               `db:"cluster"`
	Namespace   string               `db:"namespace"`
	CreatedAt   *time.Time           `db:"created_time"`
	Types       []storage.SecretType `db:"secret_type"` // Will be aggregated via array_agg
}

// toListSecret converts the database response to a storage.ListSecret protobuf.
func (r *listSecretResponse) toListSecret() *storage.ListSecret {
	types := r.Types
	if len(types) == 0 {
		types = []storage.SecretType{storage.SecretType_UNDETERMINED}
	}

	return &storage.ListSecret{
		Id:          r.ID,
		Name:        r.Name,
		ClusterId:   r.ClusterID,
		ClusterName: r.ClusterName,
		Namespace:   r.Namespace,
		CreatedAt:   protocompat.ConvertTimeToTimestampOrNil(r.CreatedAt),
		Types:       types,
	}
}
