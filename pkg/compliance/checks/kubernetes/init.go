package kubernetes

// Init registers all Kubernetes compliance checks.
// Called explicitly from pkg/compliance/checks/init.go instead of package init().
func Init() {
	registerControlPlaneConfig()
	registerEtcd()
	registerKubeletCommand()
	registerMasterAPIServer()
	registerMasterConfig()
	registerMasterControllerManager()
	registerMasterScheduler()
	registerPoliciesAdmissionControl()
	registerPoliciesGeneral()
	registerPoliciesNetworkCNI()
	registerPoliciesPodSecurity()
	registerPoliciesRBAC()
	registerPoliciesSecretsManagement()
	registerWorkerNodeConfig()
}
