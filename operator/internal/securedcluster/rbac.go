package securedcluster

//+kubebuilder:rbac:groups=platform.stackrox.io,resources=securedclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=platform.stackrox.io,resources=securedclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=platform.stackrox.io,resources=securedclusters/finalizers,verbs=update

// SecuredCluster requires broad administrative permissions, essentially all verbs on all resources. See
// https://github.com/stackrox/rox/blob/aa04df7b116571108763fda370046485a176a5f8/image/templates/helm/stackrox-secured-cluster/templates/sensor-rbac.yaml#L66-L72
// Operator's permissions must be superset of Helm and Operand permissions therefore here we configure full access.
// Ultimately, we should look for a way to automatically list only necessary permissions both in Helm charts and for the
// operator.
//+kubebuilder:rbac:groups=*,resources=*,verbs=*
