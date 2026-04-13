package client

import "k8s.io/apimachinery/pkg/runtime/schema"

// Standard Kubernetes GVR constants for use with the dynamic client.
// These replace typed client group accessors (e.g., CoreV1().Pods())
// with dynamic equivalents (e.g., dynClient.Resource(PodGVR)).
var (
	PodGVR                   = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	NodeGVR                  = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}
	NamespaceGVR             = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}
	SecretGVR                = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	ConfigMapGVR             = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	ServiceAccountGVR        = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "serviceaccounts"}
	ReplicationControllerGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "replicationcontrollers"}
	EventGVR                 = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "events"}

	DeploymentGVR  = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	DaemonSetGVR   = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}
	ReplicaSetGVR  = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"}
	StatefulSetGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}

	NetworkPolicyGVR = schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies"}

	ClusterRoleBindingGVR = schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterrolebindings"}
	RoleBindingGVR        = schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "rolebindings"}

	CronJobGVR             = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}
	CronJobBetaGVR         = schema.GroupVersionResource{Group: "batch", Version: "v1beta1", Resource: "cronjobs"}
	SubjectAccessReviewGVR = schema.GroupVersionResource{Group: "authorization.k8s.io", Version: "v1", Resource: "subjectaccessreviews"}
)
