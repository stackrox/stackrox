package central

//+kubebuilder:rbac:groups=platform.stackrox.io,resources=centrals,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=platform.stackrox.io,resources=centrals/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=platform.stackrox.io,resources=centrals/finalizers,verbs=update

// SecuredCluster RBAC makes access for the entire operator effectively */*/*. Therefore we use it for Central too.
//TODO(ROX-7373): Review and reduce this access.
//+kubebuilder:rbac:groups=*,resources=*,verbs=*
