package central

//+kubebuilder:rbac:groups=platform.stackrox.io,resources=centrals,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=platform.stackrox.io,resources=centrals/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=platform.stackrox.io,resources=centrals/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create
