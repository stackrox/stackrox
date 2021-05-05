package securedcluster

//+kubebuilder:rbac:groups=acs.openshift.io,resources=securedclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=acs.openshift.io,resources=securedclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=acs.openshift.io,resources=securedclusters/finalizers,verbs=update
