package securedcluster

//+kubebuilder:rbac:groups=platform.stackrox.io,resources=securedclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=platform.stackrox.io,resources=securedclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=platform.stackrox.io,resources=securedclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=secrets,verbs=get;list;watch
