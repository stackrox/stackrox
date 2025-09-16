package datastore

import (
	"context"

	pgStore "github.com/stackrox/rox/central/config/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	// DefaultDeployAlertRetention is the number of days to retain resolved deployment alerts.
	DefaultDeployAlertRetention = 7
	// DefaultRuntimeAlertRetention is the number of days to retain all runtime alerts.
	DefaultRuntimeAlertRetention = 30
	// DefaultDeletedRuntimeAlertRetention is the number of days to retain runtime alerts for deleted deployments.
	DefaultDeletedRuntimeAlertRetention = 7
	// DefaultImageRetention is the number of days to retain images for.
	DefaultImageRetention = 7
	// DefaultAttemptedDeployAlertRetention is the number of days to retain all attempted deploy-time alerts.
	DefaultAttemptedDeployAlertRetention = 7
	// DefaultAttemptedRuntimeAlertRetention is the number of days to retain all attempted run-time alerts.
	DefaultAttemptedRuntimeAlertRetention = 7
	// DefaultExpiredVulnReqRetention is the number of days to retain expired vulnerability requests.
	DefaultExpiredVulnReqRetention = 90
	// DefaultDecommissionedClusterRetentionDays is the number of days to retain a cluster that is unreachable.
	DefaultDecommissionedClusterRetentionDays = 0
	// DefaultReportHistoryRetentionWindow number of days to retain reports.
	DefaultReportHistoryRetentionWindow = 7
	// DefaultDownloadableReportRetentionDays number of days to retain downloadable reports.
	DefaultDownloadableReportRetentionDays = 7
	// DefaultDownloadableReportGlobalRetentionBytes is the maximum total upper limit in bytes for all downloadable reports.
	DefaultDownloadableReportGlobalRetentionBytes = 500 * 1024 * 1024
	// DefaultAdministrationEventsRetention is the number of days to retain administration events.
	DefaultAdministrationEventsRetention = 4
	// PlatformComponentSystemRuleName is the name of the system defined rule for matching openshift and kube workloads
	PlatformComponentSystemRuleName = "system rule"
	// PlatformComponentSystemRegex is the system defined regex for matching kube and openshift workloads, this is un-editable by users
	PlatformComponentSystemRegex = `^openshift$|^openshift-apiserver$|^openshift-operators$|^kube-.*`
	// PlatformComponentLayeredProductsRuleName is the name of the system defined rule for matching workloads created by Red hat layered products
	PlatformComponentLayeredProductsRuleName = "red hat layered products"
	// PlatformComponentLayeredProductsDefaultRegex is the default regex for matching workloads created by Red hat layered products, this can be edited by users
	PlatformComponentLayeredProductsDefaultRegex = `^aap$|^ack-system$|^aws-load-balancer-operator$|^cert-manager-operator$|^cert-utils-operator$|^costmanagement-metrics-operator$|^external-dns-operator$|^metallb-system$|^mtr$|^multicluster-engine$|^multicluster-global-hub$|^node-observability-operator$|^open-cluster-management$|^openshift-adp$|^openshift-apiserver-operator$|^openshift-authentication$|^openshift-authentication-operator$|^openshift-builds$|^openshift-cloud-controller-manager$|^openshift-cloud-controller-manager-operator$|^openshift-cloud-credential-operator$|^openshift-cloud-network-config-controller$|^openshift-cluster-csi-drivers$|^openshift-cluster-machine-approver$|^openshift-cluster-node-tuning-operator$|^openshift-cluster-observability-operator$|^openshift-cluster-samples-operator$|^openshift-cluster-storage-operator$|^openshift-cluster-version$|^openshift-cnv$|^openshift-compliance$|^openshift-config$|^openshift-config-managed$|^openshift-config-operator$|^openshift-console$|^openshift-console-operator$|^openshift-console-user-settings$|^openshift-controller-manager$|^openshift-controller-manager-operator$|^openshift-dbaas-operator$|^openshift-distributed-tracing$|^openshift-dns$|^openshift-dns-operator$|^openshift-dpu-network-operator$|^openshift-dr-system$|^openshift-etcd$|^openshift-etcd-operator$|^openshift-file-integrity$|^openshift-gitops-operator$|^openshift-host-network$|^openshift-image-registry$|^openshift-infra$|^openshift-ingress$|^openshift-ingress-canary$|^openshift-ingress-node-firewall$|^openshift-ingress-operator$|^openshift-insights$|^openshift-keda$|^openshift-kmm$|^openshift-kmm-hub$|^openshift-kni-infra$|^openshift-kube-apiserver$|^openshift-kube-apiserver-operator$|^openshift-kube-controller-manager$|^openshift-kube-controller-manager-operator$|^openshift-kube-scheduler$|^openshift-kube-scheduler-operator$|^openshift-kube-storage-version-migrator$|^openshift-kube-storage-version-migrator-operator$|^openshift-lifecycle-agent$|^openshift-local-storage$|^openshift-logging$|^openshift-machine-api$|^openshift-machine-config-operator$|^openshift-marketplace$|^openshift-migration$|^openshift-monitoring$|^openshift-mta$|^openshift-mtv$|^openshift-multus$|^openshift-netobserv-operator$|^openshift-network-diagnostics$|^openshift-network-node-identity$|^openshift-network-operator$|^openshift-nfd$|^openshift-nmstate$|^openshift-node$|^openshift-nutanix-infra$|^openshift-oauth-apiserver$|^openshift-openstack-infra$|^openshift-opentelemetry-operator$|^openshift-operator-lifecycle-manager$|^openshift-operators$|^openshift-operators-redhat$|^openshift-ovirt-infra$|^openshift-ovn-kubernetes$|^openshift-ptp$|^openshift-route-controller-manager$|^openshift-sandboxed-containers-operator$|^openshift-security-profiles$|^openshift-serverless$|^openshift-serverless-logic$|^openshift-service-ca$|^openshift-service-ca-operator$|^openshift-sriov-network-operator$|^openshift-storage$|^openshift-tempo-operator$|^openshift-update-service$|^openshift-user-workload-monitoring$|^openshift-vertical-pod-autoscaler$|^openshift-vsphere-infra$|^openshift-windows-machine-config-operator$|^openshift-workload-availability$|^redhat-ods-operator$|^rhdh-operator$|^service-telemetry$|^stackrox$|^submariner-operator$`
)

var (
	once sync.Once

	d DataStore

	defaultAlertRetention = &storage.PrivateConfig_AlertConfig{
		AlertConfig: &storage.AlertRetentionConfig{
			ResolvedDeployRetentionDurationDays:   DefaultDeployAlertRetention,
			DeletedRuntimeRetentionDurationDays:   DefaultDeletedRuntimeAlertRetention,
			AllRuntimeRetentionDurationDays:       DefaultRuntimeAlertRetention,
			AttemptedDeployRetentionDurationDays:  DefaultAttemptedDeployAlertRetention,
			AttemptedRuntimeRetentionDurationDays: DefaultAttemptedRuntimeAlertRetention,
		},
	}

	defaultPrivateConfig = storage.PrivateConfig{
		ImageRetentionDurationDays:          DefaultImageRetention,
		AlertRetention:                      defaultAlertRetention,
		ExpiredVulnReqRetentionDurationDays: DefaultExpiredVulnReqRetention,
	}

	defaultVulnerabilityDeferralConfig = &storage.VulnerabilityExceptionConfig{
		ExpiryOptions: &storage.VulnerabilityExceptionConfig_ExpiryOptions{
			DayOptions: []*storage.DayOption{
				{
					NumDays: 14,
					Enabled: true,
				},
				{
					NumDays: 30,
					Enabled: true,
				},
				{
					NumDays: 60,
					Enabled: true,
				},
				{
					NumDays: 90,
					Enabled: true,
				},
			},
			FixableCveOptions: &storage.VulnerabilityExceptionConfig_FixableCVEOptions{
				AllFixable: true,
				AnyFixable: true,
			},
			CustomDate: false,
			Indefinite: false,
		},
	}

	defaultDecommissionedClusterRetention = &storage.DecommissionedClusterRetentionConfig{
		RetentionDurationDays: DefaultDecommissionedClusterRetentionDays,
	}

	defaultReportRetentionConfig = &storage.ReportRetentionConfig{
		HistoryRetentionDurationDays:           DefaultReportHistoryRetentionWindow,
		DownloadableReportRetentionDays:        DefaultDownloadableReportRetentionDays,
		DownloadableReportGlobalRetentionBytes: DefaultDownloadableReportGlobalRetentionBytes,
	}

	defaultAdministrationEventsConfig = &storage.AdministrationEventsConfig{
		RetentionDurationDays: DefaultAdministrationEventsRetention,
	}

	defaultPlatformConfigSystemRule = &storage.PlatformComponentConfig_Rule{
		Name: PlatformComponentSystemRuleName,
		NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
			Regex: PlatformComponentSystemRegex,
		},
	}
	defaultPlatformConfigLayeredProductsRule = &storage.PlatformComponentConfig_Rule{
		Name: PlatformComponentLayeredProductsRuleName,
		NamespaceRule: &storage.PlatformComponentConfig_Rule_NamespaceRule{
			Regex: PlatformComponentLayeredProductsDefaultRegex,
		},
	}
)

func validateConfigAndPopulateMissingDefaults(datastore DataStore) {
	ctx := sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	config, err := datastore.GetConfig(ctx)
	if err != nil {
		panic(err)
	}
	if config == nil {
		config = &storage.Config{}
	}

	// See the note next to the publicConfigCache variable in datastore.go for
	// more information on public config caching.
	cachePublicConfig(config.GetPublicConfig())

	needsUpsert := false
	privateConfig := config.GetPrivateConfig()
	if privateConfig == nil {
		privateConfig = defaultPrivateConfig.CloneVT()
		needsUpsert = true
	}

	if privateConfig.GetDecommissionedClusterRetention() == nil {
		privateConfig.DecommissionedClusterRetention = defaultDecommissionedClusterRetention
		needsUpsert = true
	}

	if privateConfig.GetReportRetentionConfig() == nil {
		privateConfig.ReportRetentionConfig = defaultReportRetentionConfig
		needsUpsert = true
	}

	if features.UnifiedCVEDeferral.Enabled() {
		if privateConfig.GetVulnerabilityExceptionConfig() == nil {
			privateConfig.VulnerabilityExceptionConfig = defaultVulnerabilityDeferralConfig
			needsUpsert = true
		}
	}

	if privateConfig.GetAdministrationEventsConfig() == nil {
		privateConfig.AdministrationEventsConfig = defaultAdministrationEventsConfig
		needsUpsert = true
	}

	if features.CustomizablePlatformComponents.Enabled() && populateDefaultSystemRulesIfMissing(config) {
		needsUpsert = true
	}

	if needsUpsert {
		config.PrivateConfig = privateConfig
		utils.Must(datastore.UpsertConfig(ctx, config))
	}
}

// populateDefaultSystemRuleIfMissing initializes platform component config's system and layered products rules if they are not already initialized.
// It returns true if either of the rules were initialized
func populateDefaultSystemRulesIfMissing(config *storage.Config) bool {
	// If there is no platform component config, initialize both system and layered product rules and return
	if config.GetPlatformComponentConfig() == nil {
		config.PlatformComponentConfig = &storage.PlatformComponentConfig{
			Rules: []*storage.PlatformComponentConfig_Rule{
				defaultPlatformConfigSystemRule,
				defaultPlatformConfigLayeredProductsRule,
			},
			NeedsReevaluation: true,
		}
		return true
	}
	hasSystemRule := false
	hasLayeredProductsRule := false
	for _, rule := range config.GetPlatformComponentConfig().GetRules() {
		if rule.GetName() == PlatformComponentSystemRuleName {
			hasSystemRule = true
		} else if rule.GetName() == PlatformComponentLayeredProductsRuleName {
			hasLayeredProductsRule = true
		}
	}

	if hasSystemRule && hasLayeredProductsRule {
		// If both system and layered products rules exist, no need to initialize anything
		return false
	}

	if !hasSystemRule {
		// if system rule is missing, initialize it
		config.GetPlatformComponentConfig().Rules = append(
			config.GetPlatformComponentConfig().Rules,
			defaultPlatformConfigSystemRule,
		)
		config.GetPlatformComponentConfig().NeedsReevaluation = true
	}
	if !hasLayeredProductsRule {
		// if layered products rule is missing, initialize it
		config.GetPlatformComponentConfig().Rules = append(
			config.GetPlatformComponentConfig().Rules,
			defaultPlatformConfigLayeredProductsRule,
		)
		config.GetPlatformComponentConfig().NeedsReevaluation = true
	}
	return true
}

func initialize() {
	store := pgStore.New(globaldb.GetPostgres())

	d = New(store)

	validateConfigAndPopulateMissingDefaults(d)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return d
}
