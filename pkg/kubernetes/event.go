package kubernetes

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
)

// EventAsString returns the kubernetes resources as string, such as, namespace/default/pod/nginx-86c57db685-nqq97/portforward.
func EventAsString(event *storage.KubernetesEvent) string {
	resource, subresource := stringutils.Split2(strings.ToLower(event.GetObject().GetResource().String()), "_")
	suffix := resource + "/" + event.GetObject().GetName()
	if subresource != "" {
		suffix = suffix + "/" + subresource
	}

	if event.GetObject().GetScopeInfo() == nil {
		return suffix
	}

	var prefix string
	if event.GetObject().GetDeploymentScopeInfo() != nil {
		prefix = "namespace/" + event.GetObject().GetDeploymentScopeInfo().GetNamespace()
	}

	if event.GetObject().GetNamespaceScopeInfo() != nil {
		prefix = "namespace/" + event.GetObject().GetNamespaceScopeInfo().GetNamespace()
	}

	return event.GetApiVerb().String() + "/" + prefix + "/" + suffix
}
