package securedcluster

//+kubebuilder:rbac:groups=platform.stackrox.io,resources=securedclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=platform.stackrox.io,resources=securedclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=platform.stackrox.io,resources=securedclusters/finalizers,verbs=update

// The following permission is added because of this
// https://github.com/stackrox/rox/blob/aa04df7b116571108763fda370046485a176a5f8/image/templates/helm/stackrox-secured-cluster/templates/sensor-rbac.yaml#L66-L72
// We should review access that helm chart assigns, reduce it and reduce RBAC claims here.
// TODO(ROX-7373): Review and reduce this access.
//+kubebuilder:rbac:groups=*,resources=*,verbs=*
