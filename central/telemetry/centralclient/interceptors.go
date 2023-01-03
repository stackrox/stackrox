package centralclient

import (
	"net/http"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var (
	trackedPaths []string
	ignoredPaths = []string{"/v1/ping", "/v1.PingService/Ping", "/v1/metadata", "/static/*"}

	uninitializedClusters     = set.NewSet[string]()
	uninitializedClustersLock = &sync.Mutex{}

	interceptors = map[string][]phonehome.Interceptor{
		"API Call":            {apiCall, addDefaultProps},
		"Cluster Registered":  {clusterRegistered},
		"Cluster Initialized": {clusterInitialized},
		"roxctl":              {roxctl, addDefaultProps},

		"Create Auth Provider":  {createAuthProvider, addDefaultProps},
		"Create Access Scope":   {createSimpleAccessScope, addDefaultProps},
		"Create Permission Set": {createPermissionSet, addDefaultProps},
		"Create Role":           {createRole, addDefaultProps},
	}
)

func addDefaultProps(rp *phonehome.RequestParams, props map[string]any) bool {
	props["Path"] = rp.Path
	props["Code"] = rp.Code
	props["Method"] = rp.Method
	props["User-Agent"] = rp.UserAgent
	return true
}

// apiCall enables API Call events for the API paths specified in the
// trackedPaths ("*" value enables all paths) and have no match in the
// ignoredPaths list.
func apiCall(rp *phonehome.RequestParams, props map[string]any) bool {
	return !rp.HasPathIn(ignoredPaths) && rp.HasPathIn(trackedPaths)
}

var postCluster = &phonehome.ServiceMethod{
	GRPCMethod: "/v1.ClustersService/PostCluster",
	HTTPMethod: http.MethodPost,
	HTTPPath:   "/v1/cluster",
}

// clusterRegistered enables the Cluster Registered event and adds specific
// properties.
func clusterRegistered(rp *phonehome.RequestParams, props map[string]any) bool {
	if !rp.Is(postCluster) {
		return false
	}

	props["Code"] = rp.Code
	if cluster := phonehome.GetGRPCRequestBody(v1.ClustersServiceServer.PostCluster, rp); cluster != nil {
		props["Cluster Type"] = cluster.GetType().String()
		props["Cluster ID"] = cluster.GetId()
		props["Managed By"] = cluster.GetManagedBy().String()
		uninitializedClustersLock.Lock()
		defer uninitializedClustersLock.Unlock()
		if cluster.GetHealthStatus().GetSensorHealthStatus() == storage.ClusterHealthStatus_UNINITIALIZED {
			uninitializedClusters.Add(cluster.GetId())
		}
	}
	return true
}

var putCluster = &phonehome.ServiceMethod{
	GRPCMethod: "/v1.ClustersService/PutCluster",
	HTTPMethod: http.MethodPut,
	HTTPPath:   "/v1/cluster/*",
}

// clusterInitialized enables the Cluster Initialized event and adds specific
// properties.
func clusterInitialized(rp *phonehome.RequestParams, props map[string]any) bool {
	if !rp.Is(putCluster) {
		return false
	}

	if cluster := phonehome.GetGRPCRequestBody(v1.ClustersServiceServer.PutCluster, rp); cluster != nil {
		uninitializedClustersLock.Lock()
		defer uninitializedClustersLock.Unlock()

		newStatus := cluster.GetHealthStatus().GetSensorHealthStatus()
		if newStatus == storage.ClusterHealthStatus_UNINITIALIZED {
			uninitializedClusters.Add(cluster.GetId())
		} else
		// Fire an event if the sensor moves from UNINITIALIZED state.
		// The event will be missed if the central restarts between
		// postCluster and first putCluster.
		if uninitializedClusters.Contains(cluster.GetId()) &&
			newStatus != storage.ClusterHealthStatus_UNINITIALIZED {
			uninitializedClusters.Remove(cluster.GetId())
			props["Code"] = rp.Code
			props["Cluster Type"] = cluster.GetType().String()
			props["Cluster ID"] = cluster.GetId()
			props["Managed By"] = cluster.GetManagedBy().String()
			return true
		}
	}
	return false
}

// roxctl enables the roxctl event.
func roxctl(rp *phonehome.RequestParams, props map[string]any) bool {
	return strings.Contains(rp.UserAgent, "roxctl")
}

//
// Access control.
//

// Auth Provider:

var postAuthProvider = &phonehome.ServiceMethod{
	GRPCMethod: "/v1.AuthProviderService/PostAuthProvider",
	HTTPMethod: http.MethodPost,
	HTTPPath:   "/v1/authProviders",
}

var putAuthProvider = &phonehome.ServiceMethod{
	GRPCMethod: "/v1.AuthProviderService/PutAuthProvider",
	HTTPMethod: http.MethodPut,
	HTTPPath:   "/v1/authProviders/*",
}

func createAuthProvider(rp *phonehome.RequestParams, props map[string]any) bool {
	switch {
	case rp.Is(postAuthProvider):
		if ap, err := phonehome.GetGRPCRequestBody(v1.AuthProviderServiceServer.PostAuthProvider, rp); err == nil {
			props["Type"] = ap.GetProvider().GetType()
		}
		return true
	case rp.Is(putAuthProvider):
		if ap, err := phonehome.GetGRPCRequestBody(v1.AuthProviderServiceServer.PutAuthProvider, rp); err == nil {
			props["Type"] = ap.GetType()
		}
		return true
	}
	return false
}

// Simple Access Scope:

var postSimpleAccessScope = &phonehome.ServiceMethod{
	GRPCMethod: "/v1.RoleService/PostSimpleAccessScope",
	HTTPMethod: http.MethodPost,
	HTTPPath:   "/v1/simpleaccessscopes",
}

var putSimpleAccessScope = &phonehome.ServiceMethod{
	GRPCMethod: "/v1.RoleService/PutSimpleAccessScope",
	HTTPMethod: http.MethodPut,
	HTTPPath:   "/v1/simpleaccessscopes/*",
}

func createSimpleAccessScope(rp *phonehome.RequestParams, props map[string]any) bool {
	return rp.Is(postSimpleAccessScope) || rp.Is(putSimpleAccessScope)
}

// Permission Set:

var postPermissionSet = &phonehome.ServiceMethod{
	GRPCMethod: "/v1.RoleService/PostPermissionSet",
	HTTPMethod: http.MethodPost,
	HTTPPath:   "/v1/permissionsets",
}

var putPermissionSet = &phonehome.ServiceMethod{
	GRPCMethod: "/v1.RoleService/PutPermissionSet",
	HTTPMethod: http.MethodPut,
	HTTPPath:   "/v1/permissionsets/*",
}

func createPermissionSet(rp *phonehome.RequestParams, props map[string]any) bool {
	return rp.Is(postPermissionSet) || rp.Is(putPermissionSet)
}

// Role:

var postRole = &phonehome.ServiceMethod{
	GRPCMethod: "/v1.RoleService/CreateRole",
	HTTPMethod: http.MethodPost,
	HTTPPath:   "/v1/roles",
}

var putRole = &phonehome.ServiceMethod{
	GRPCMethod: "/v1.RoleService/UpdateRole",
	HTTPMethod: http.MethodPut,
	HTTPPath:   "/v1/roles/*",
}

func createRole(rp *phonehome.RequestParams, props map[string]any) bool {
	return rp.Is(postRole) || rp.Is(putRole)
}
