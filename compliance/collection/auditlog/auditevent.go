package auditlog

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
)

const (
	reasonAnnotationKey = "authorization.k8s.io/reason"
)

var (
	// The audit logs report the resource all as one word, but the k8s event object (and elsewhere) uses underscore
	auditResourceToKubeResource = map[string]storage.KubernetesEvent_Object_Resource{
		"pods_exec":                  storage.KubernetesEvent_Object_PODS_EXEC,
		"pods_portforward":           storage.KubernetesEvent_Object_PODS_PORTFORWARD,
		"secrets":                    storage.KubernetesEvent_Object_SECRETS,
		"configmaps":                 storage.KubernetesEvent_Object_CONFIGMAPS,
		"clusterroles":               storage.KubernetesEvent_Object_CLUSTER_ROLES,
		"clusterrolebindings":        storage.KubernetesEvent_Object_CLUSTER_ROLE_BINDINGS,
		"networkpolicies":            storage.KubernetesEvent_Object_NETWORK_POLICIES,
		"securitycontextconstraints": storage.KubernetesEvent_Object_SECURITY_CONTEXT_CONSTRAINTS,
		"egressfirewalls":            storage.KubernetesEvent_Object_EGRESS_FIREWALLS,
	}
)

type auditEvent struct {
	Annotations              map[string]string `json:"annotations"`
	APIVersion               string            `json:"apiVersion"`
	AuditID                  string            `json:"auditID"`
	Kind                     string            `json:"kind"`
	Level                    string            `json:"level"`
	ObjectRef                objectRef         `json:"objectRef"`
	RequestReceivedTimestamp string            `json:"requestReceivedTimestamp"`
	RequestURI               string            `json:"requestURI"`
	ResponseStatus           responseStatusRef `json:"responseStatus"`
	SourceIPs                []string          `json:"sourceIPs"`
	Stage                    string            `json:"stage"`
	StageTimestamp           string            `json:"stageTimestamp"`
	User                     userRef           `json:"user"`
	ImpersonatedUser         *userRef          `json:"impersonatedUser,omitempty"`
	UserAgent                string            `json:"userAgent"`
	Verb                     string            `json:"verb"`
}

type objectRef struct {
	APIVersion string `json:"apiVersion"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Resource   string `json:"resource"`
}

type userRef struct {
	Username string   `json:"username"`
	UID      string   `json:"uid"`
	Groups   []string `json:"groups"`
}

type responseStatusRef struct {
	Metadata map[string]interface{} `json:"metadata"`
	Status   string                 `json:"status"`
	Message  string                 `json:"message"`
	Code     int32                  `json:"code"`
}

func (u *userRef) ToKubernetesEventUser() *storage.KubernetesEvent_User {
	ku := &storage.KubernetesEvent_User{}
	ku.SetUsername(u.Username)
	ku.SetGroups(u.Groups)
	return ku
}

func (e *auditEvent) ToKubernetesEvent(clusterID string) *storage.KubernetesEvent {
	protoTime, err := protocompat.ParseRFC3339NanoTimestamp(e.StageTimestamp)
	if err != nil {
		log.Errorf("Failed to parse stage time %s from audit log, so falling back to received time: %v", e.StageTimestamp, err)
		// If StageTimestamp (which is the time for this particular stage) is not parsable, try the RequestReceivedTimestamp
		// While it's not as accurate it should be relatively close. This should also be a rare occurrence.
		protoTime, err = protocompat.ParseRFC3339NanoTimestamp(e.RequestReceivedTimestamp)
		if err != nil {
			protoTime = nil
		}
	}
	if err != nil {
		// If we're still not able to get a valid time, fall back to "now".
		log.Errorf(
			"Failed to parse received time %s from audit log for event %s:%s/%s/%s, so falling back to current time. Error: %v",
			e.RequestReceivedTimestamp,
			e.Verb,
			e.ObjectRef.Namespace,
			e.ObjectRef.Resource,
			e.ObjectRef.Name,
			err)
		protoTime = protocompat.TimestampNow()
	}

	reason := e.Annotations[reasonAnnotationKey]

	resource, found := auditResourceToKubeResource[strings.ToLower(e.ObjectRef.Resource)]
	if !found {
		resource = storage.KubernetesEvent_Object_UNKNOWN
	}

	ko := &storage.KubernetesEvent_Object{}
	ko.SetName(e.ObjectRef.Name)
	ko.SetResource(resource)
	ko.SetClusterId(clusterID)
	ko.SetNamespace(e.ObjectRef.Namespace)
	kr := &storage.KubernetesEvent_ResponseStatus{}
	kr.SetStatusCode(e.ResponseStatus.Code)
	kr.SetReason(reason)
	k8sEvent := &storage.KubernetesEvent{}
	k8sEvent.SetId(e.AuditID)
	k8sEvent.SetObject(ko)
	k8sEvent.SetTimestamp(protoTime)
	k8sEvent.SetApiVerb(storage.KubernetesEvent_APIVerb(storage.KubernetesEvent_APIVerb_value[strings.ToUpper(e.Verb)]))
	k8sEvent.SetUser(e.User.ToKubernetesEventUser())
	k8sEvent.SetSourceIps(e.SourceIPs)
	k8sEvent.SetUserAgent(e.UserAgent)
	k8sEvent.SetResponseStatus(kr)
	k8sEvent.SetRequestUri(e.RequestURI)

	if e.ImpersonatedUser != nil {
		k8sEvent.SetImpersonatedUser(e.ImpersonatedUser.ToKubernetesEventUser())
	}

	return k8sEvent
}
