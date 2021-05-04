package central

//+kubebuilder:rbac:groups=acs.openshift.io,resources=centrals,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=acs.openshift.io,resources=centrals/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=acs.openshift.io,resources=centrals/finalizers,verbs=update
