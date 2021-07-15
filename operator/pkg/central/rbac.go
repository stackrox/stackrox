package central

//+kubebuilder:rbac:groups=platform.stackrox.io,resources=centrals,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=platform.stackrox.io,resources=centrals/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=platform.stackrox.io,resources=centrals/finalizers,verbs=update

// SecuredCluster RBAC is */*/* (see `../securecluster/rbac.go`), and, since both controllers are packaged in the single
// operator, it does not make practical sense to configure more narrow RBAC for Central.
// Therefore, Central RBAC is configured the same as for SecuredCluster.
// This is not optimal from security point of view; see notes for SecuredCluster how this could be improved.
//+kubebuilder:rbac:groups=*,resources=*,verbs=*
