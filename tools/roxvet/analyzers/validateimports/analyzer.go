package validateimports

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

const doc = `check that imports are valid`

const roxPrefix = "github.com/stackrox/rox/"

var (
	// Keep this in alphabetic order.
	validRoots = set.NewFrozenStringSet(
		"central",
		"compliance",
		"govulncheck",
		"image",
		"migrator",
		"migrator/migrations",
		"operator",
		"pkg",
		"roxctl",
		"scale",
		"scanner",
		"sensor/admission-control",
		"sensor/common",
		"sensor/debugger",
		"sensor/kubernetes",
		"sensor/tests",
		"sensor/testutils",
		"sensor/upgrader",
		"sensor/utils",
		"tools",
		"webhookserver",
		"qa-tests-backend/test-images/syslog",
	)

	ignoredRoots = []string{
		"generated",
		"tests",
		"local",
	}

	forbiddenImports = map[string]struct {
		replacement string
		allowlist   set.StringSet
	}{
		"io/ioutil": {
			replacement: "https://golang.org/doc/go1.18#ioutil",
		},
		"sync": {
			replacement: "github.com/stackrox/rox/pkg/sync",
			allowlist: set.NewStringSet(
				"github.com/stackrox/rox/pkg/bolthelper/crud/proto",
			),
		},
		"github.com/gogo/protobuf/proto": {
			replacement: "pkg/proto*",
			allowlist: set.NewStringSet(
				"github.com/stackrox/rox/pkg/protocompat",
				"github.com/stackrox/rox/pkg/protoconv",
				"github.com/stackrox/rox/pkg/protoutils",
				// The packages below should be removed from the set
				// once migrated to the compatibility layer (above three packages).
				"github.com/stackrox/rox/central/audit",
				"github.com/stackrox/rox/central/cluster/datastore",
				"github.com/stackrox/rox/central/compliance/checks/common",
				"github.com/stackrox/rox/central/cve/edgefields",
				"github.com/stackrox/rox/central/declarativeconfig",
				"github.com/stackrox/rox/central/declarativeconfig/types",
				"github.com/stackrox/rox/central/declarativeconfig/updater",
				"github.com/stackrox/rox/central/declarativeconfig/updater/mocks",
				"github.com/stackrox/rox/central/declarativeconfig/utils",
				"github.com/stackrox/rox/central/detection/alertmanager",
				"github.com/stackrox/rox/central/detection/buildtime",
				"github.com/stackrox/rox/central/detection/lifecycle",
				"github.com/stackrox/rox/central/globaldb/v2backuprestore/service",
				"github.com/stackrox/rox/central/group/datastore",
				"github.com/stackrox/rox/central/group/service",
				"github.com/stackrox/rox/central/metadata/service",
				"github.com/stackrox/rox/central/networkpolicies/generator",
				"github.com/stackrox/rox/central/networkpolicies/graph",
				"github.com/stackrox/rox/central/notifiers/generic",
				"github.com/stackrox/rox/central/notifiers/splunk",
				"github.com/stackrox/rox/central/notifiers/sumologic",
				"github.com/stackrox/rox/central/probeupload/service",
				"github.com/stackrox/rox/central/processlisteningonport/store/postgres",
				"github.com/stackrox/rox/central/risk/manager",
				"github.com/stackrox/rox/central/role/mapper",
				"github.com/stackrox/rox/central/scrape",
				"github.com/stackrox/rox/central/sensor/service/pipeline/all",
				"github.com/stackrox/rox/central/sensor/service/pipeline/networkflowupdate",
				"github.com/stackrox/rox/central/version",
				"github.com/stackrox/rox/migrator/migrations/dackboxhelpers/activecomponent",
				"github.com/stackrox/rox/migrator/migrations/dackboxhelpers/componentcveedge",
				"github.com/stackrox/rox/migrator/migrations/dackboxhelpers/cve",
				"github.com/stackrox/rox/migrator/migrations/dackboxhelpers/deployment",
				"github.com/stackrox/rox/migrator/migrations/dackboxhelpers/image",
				"github.com/stackrox/rox/migrator/migrations/dackboxhelpers/imagecomponent",
				"github.com/stackrox/rox/migrator/migrations/dackboxhelpers/imagecomponentedge",
				"github.com/stackrox/rox/migrator/migrations/dackboxhelpers/imagecveedge",
				"github.com/stackrox/rox/migrator/migrations/dackboxhelpers/node",
				"github.com/stackrox/rox/migrator/migrations/dackboxhelpers/nodecomponentedge",
				"github.com/stackrox/rox/migrator/migrations/m_100_to_m_101_cluster_id_netpol_undo_store",
				"github.com/stackrox/rox/migrator/migrations/m_102_to_m_103_migrate_serial",
				"github.com/stackrox/rox/migrator/migrations/m_104_to_m_105_active_component",
				"github.com/stackrox/rox/migrator/migrations/m_105_to_m_106_group_id",
				"github.com/stackrox/rox/migrator/migrations/m_106_to_m_107_policy_categories",
				"github.com/stackrox/rox/migrator/migrations/m_107_to_m_108_remove_auth_plugin",
				"github.com/stackrox/rox/migrator/migrations/m_108_to_m_109_compliance_run_schedules",
				"github.com/stackrox/rox/migrator/migrations/m_110_to_m_111_replace_deprecated_resources",
				"github.com/stackrox/rox/migrator/migrations/m_111_to_m_112_groups_invalid_values",
				"github.com/stackrox/rox/migrator/migrations/m_56_to_m_57_compliance_policy_categories",
				"github.com/stackrox/rox/migrator/migrations/m_57_to_m_58_update_run_secrets_volume_policy_regex",
				"github.com/stackrox/rox/migrator/migrations/m_58_to_m_59_node_scanning_flag_on",
				"github.com/stackrox/rox/migrator/migrations/m_59_to_m_60_add_docker_cis_category_to_existing",
				"github.com/stackrox/rox/migrator/migrations/m_60_to_m_61_update_network_management_policy_regex",
				"github.com/stackrox/rox/migrator/migrations/m_61_to_m_62_multiple_cve_types",
				"github.com/stackrox/rox/migrator/migrations/m_62_to_m_63_splunk_source_type",
				"github.com/stackrox/rox/migrator/migrations/m_63_to_m_64_exclude_some_openshift_operators_from_policies",
				"github.com/stackrox/rox/migrator/migrations/m_64_to_m_65_detect_openshift4_cluster_on_exec_webhooks",
				"github.com/stackrox/rox/migrator/migrations/m_65_to_m_66_policy_bug_fixes",
				"github.com/stackrox/rox/migrator/migrations/m_66_to_m_67_missing_policy_migrations",
				"github.com/stackrox/rox/migrator/migrations/m_67_to_m_68_exclude_pdcsi_from_mount_propagation",
				"github.com/stackrox/rox/migrator/migrations/m_68_to_m_69_update_global_access_roles",
				"github.com/stackrox/rox/migrator/migrations/m_69_to_m_70_add_xmrig_to_crypto_policy",
				"github.com/stackrox/rox/migrator/migrations/m_70_to_m_71_disable_audit_log_collection",
				"github.com/stackrox/rox/migrator/migrations/m_72_to_m_73_change_roles_to_sac_v2",
				"github.com/stackrox/rox/migrator/migrations/m_73_to_m_74_runtime_policy_event_source",
				"github.com/stackrox/rox/migrator/migrations/m_74_to_m_75_severity_policy",
				"github.com/stackrox/rox/migrator/migrations/m_75_to_m_76_exclude_compliance_operator_dnf_policy",
				"github.com/stackrox/rox/migrator/migrations/m_76_to_m_77_move_roles_to_rocksdb",
				"github.com/stackrox/rox/migrator/migrations/m_77_to_m_78_mitre",
				"github.com/stackrox/rox/migrator/migrations/m_80_to_m_81_rm_demo_policies",
				"github.com/stackrox/rox/migrator/migrations/m_82_to_m_83_default_pol_flag",
				"github.com/stackrox/rox/migrator/migrations/m_89_to_m_90_vuln_state",
				"github.com/stackrox/rox/migrator/migrations/m_90_to_m_91_snooze_permissions",
				"github.com/stackrox/rox/migrator/migrations/m_91_to_m_92_write_edges_to_graph",
				"github.com/stackrox/rox/migrator/migrations/m_92_to_m_93_cleanup_orphaned_rbac_cluster_objs",
				"github.com/stackrox/rox/migrator/migrations/m_93_to_m_94_role_accessscopeid",
				"github.com/stackrox/rox/migrator/migrations/m_94_to_m_95_cluster_health_status_id",
				"github.com/stackrox/rox/migrator/migrations/m_95_to_m_96_alert_scoping_information_at_root",
				"github.com/stackrox/rox/migrator/migrations/m_96_to_m_97_modify_default_vulnreportcreator_role",
				"github.com/stackrox/rox/migrator/migrations/n_01_to_n_02_postgres_clusters/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_02_to_n_03_postgres_namespaces/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_03_to_n_04_postgres_deployments/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_04_to_n_05_postgres_images/postgres",
				"github.com/stackrox/rox/migrator/migrations/n_05_to_n_06_postgres_active_components/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_06_to_n_07_postgres_alerts/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_07_to_n_08_postgres_api_tokens/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_08_to_n_09_postgres_auth_providers/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_09_to_n_10_postgres_cluster_cves/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_10_to_n_11_postgres_cluster_health_statuses",
				"github.com/stackrox/rox/migrator/migrations/n_10_to_n_11_postgres_cluster_health_statuses/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_11_to_n_12_postgres_cluster_init_bundles/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_13_to_n_14_postgres_compliance_operator_check_results/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_14_to_n_15_postgres_compliance_operator_profiles/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_15_to_n_16_postgres_compliance_operator_rules/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_16_to_n_17_postgres_compliance_operator_scan_setting_bindings/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_17_to_n_18_postgres_compliance_operator_scans/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_21_to_n_22_postgres_configs/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_22_to_n_23_postgres_external_backups/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_23_to_n_24_postgres_image_integrations/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_24_to_n_25_postgres_installation_infos/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_25_to_n_26_postgres_integration_healths/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_26_to_n_27_postgres_k8s_roles/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_28_to_n_29_postgres_network_baselines/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_29_to_n_30_postgres_network_entities/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_31_to_n_32_postgres_network_graph_configs/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_32_to_n_33_postgres_networkpolicies/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_33_to_n_34_postgres_networkpoliciesundodeployments/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_34_to_n_35_postgres_networkpolicyapplicationundorecords/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_35_to_n_36_postgres_nodes/postgres",
				"github.com/stackrox/rox/migrator/migrations/n_36_to_n_37_postgres_notifiers/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_38_to_n_39_postgres_pods/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_39_to_n_40_postgres_policies/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_40_to_n_41_postgres_process_baseline_results/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_41_to_n_42_postgres_process_baselines/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_42_to_n_43_postgres_process_indicators/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_43_to_n_44_postgres_report_configurations/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_44_to_n_45_postgres_risks/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_45_to_n_46_postgres_role_bindings/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_47_to_n_48_postgres_secrets/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_48_to_n_49_postgres_sensor_upgrade_configs/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_49_to_n_50_postgres_service_accounts/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_50_to_n_51_postgres_service_identities/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_51_to_n_52_postgres_signature_integrations/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/legacypermissionsets",
				"github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/legacyroles",
				"github.com/stackrox/rox/migrator/migrations/n_52_to_n_53_postgres_simple_access_scopes/legacysimpleaccessscopes",
				"github.com/stackrox/rox/migrator/migrations/n_53_to_n_54_postgres_vulnerability_requests/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_54_to_n_55_postgres_watched_images/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_55_to_n_56_postgres_policy_categories/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_56_to_n_57_postgres_groups/legacy",
				"github.com/stackrox/rox/migrator/migrations/policymigrationhelper",
				"github.com/stackrox/rox/migrator/rockshelper",
				"github.com/stackrox/rox/migrator/runner",
				"github.com/stackrox/rox/pkg/auth/user",
				"github.com/stackrox/rox/pkg/bolthelper/crud/proto",
				"github.com/stackrox/rox/pkg/bolthelper/singletonstore",
				"github.com/stackrox/rox/pkg/booleanpolicy",
				"github.com/stackrox/rox/pkg/dackbox",
				"github.com/stackrox/rox/pkg/dackbox/crud",
				"github.com/stackrox/rox/pkg/dackbox/indexer",
				"github.com/stackrox/rox/pkg/dackbox/indexer/mocks",
				"github.com/stackrox/rox/pkg/dackbox/indexer/tests",
				"github.com/stackrox/rox/pkg/dackbox/utils/queue",
				"github.com/stackrox/rox/pkg/dackbox/utils/queue/mocks",
				"github.com/stackrox/rox/pkg/db",
				"github.com/stackrox/rox/pkg/db/mapcache",
				"github.com/stackrox/rox/pkg/db/mocks",
				"github.com/stackrox/rox/pkg/declarativeconfig/transform",
				"github.com/stackrox/rox/pkg/declarativeconfig/transform/mocks",
				"github.com/stackrox/rox/pkg/grpc/common/authn/servicecerttoken",
				"github.com/stackrox/rox/pkg/notifier",
				"github.com/stackrox/rox/pkg/postgres/pgutils",
				"github.com/stackrox/rox/pkg/rocksdb/crud",
				"github.com/stackrox/rox/pkg/search",
				"github.com/stackrox/rox/pkg/search/postgres",
				"github.com/stackrox/rox/pkg/signatures",
				"github.com/stackrox/rox/roxctl/central/db/restore",
				"github.com/stackrox/rox/roxctl/collector/supportpackages/upload",
				"github.com/stackrox/rox/sensor/admission-control/settingswatch",
				"github.com/stackrox/rox/sensor/common/compliance",
				"github.com/stackrox/rox/sensor/common/enforcer",
				"github.com/stackrox/rox/sensor/common/sensor/helmconfig",
				"github.com/stackrox/rox/sensor/kubernetes/admissioncontroller",
				"github.com/stackrox/rox/sensor/kubernetes/networkpolicies",
				"github.com/stackrox/rox/sensor/kubernetes/upgrade",
				"github.com/stackrox/rox/tests",
				"github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings",
				"github.com/stackrox/rox/tools/rocksdbdump",
			),
		},
		"github.com/gogo/protobuf/types": {
			replacement: "pkg/proto*",
			allowlist: set.NewStringSet(
				"github.com/stackrox/rox/pkg/protocompat",
				"github.com/stackrox/rox/pkg/protoconv",
				"github.com/stackrox/rox/pkg/protoconv/resources",
				"github.com/stackrox/rox/pkg/protoutils",
				// The packages below should be removed from the set
				// once migrated to the compatibility layer (above three packages).
				"github.com/stackrox/rox/central/administration/events/service",
				"github.com/stackrox/rox/central/administration/usage/csv",
				"github.com/stackrox/rox/central/administration/usage/datastore/securedunits",
				"github.com/stackrox/rox/central/administration/usage/service",
				"github.com/stackrox/rox/central/administration/usage/store/cache",
				"github.com/stackrox/rox/central/alert/datastore",
				"github.com/stackrox/rox/central/alert/service",
				"github.com/stackrox/rox/central/apitoken/expiration",
				"github.com/stackrox/rox/central/audit",
				"github.com/stackrox/rox/central/auth/service",
				"github.com/stackrox/rox/central/blob/datastore",
				"github.com/stackrox/rox/central/blob/datastore/store",
				"github.com/stackrox/rox/central/blob/snapshot",
				"github.com/stackrox/rox/central/cluster/datastore",
				"github.com/stackrox/rox/central/cluster/service",
				"github.com/stackrox/rox/central/clusterinit/backend",
				"github.com/stackrox/rox/central/compliance/checks/hipaa_164/check308a1iia",
				"github.com/stackrox/rox/central/compliance/checks/nist80053/check_si_4",
				"github.com/stackrox/rox/central/compliance/datastore/internal/store",
				"github.com/stackrox/rox/central/compliance/manager",
				"github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore",
				"github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager",
				"github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore",
				"github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/service",
				"github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore",
				"github.com/stackrox/rox/central/config/datastore",
				"github.com/stackrox/rox/central/convert/testutils",
				"github.com/stackrox/rox/central/convert/v2tostorage",
				"github.com/stackrox/rox/central/credentialexpiry/service",
				"github.com/stackrox/rox/central/cve/cluster/datastore",
				"github.com/stackrox/rox/central/cve/cluster/datastore/mocks",
				"github.com/stackrox/rox/central/cve/cluster/datastore/store/postgres",
				"github.com/stackrox/rox/central/cve/cluster/datastoretest",
				"github.com/stackrox/rox/central/cve/cluster/service",
				"github.com/stackrox/rox/central/cve/common",
				"github.com/stackrox/rox/central/cve/fetcher",
				"github.com/stackrox/rox/central/cve/image/datastore",
				"github.com/stackrox/rox/central/cve/image/datastore/mocks",
				"github.com/stackrox/rox/central/cve/image/service",
				"github.com/stackrox/rox/central/cve/node/datastore",
				"github.com/stackrox/rox/central/cve/node/datastore/mocks",
				"github.com/stackrox/rox/central/cve/node/service",
				"github.com/stackrox/rox/central/debug/service",
				"github.com/stackrox/rox/central/declarativeconfig/health/datastore",
				"github.com/stackrox/rox/central/declarativeconfig/utils",
				"github.com/stackrox/rox/central/deployment/queue",
				"github.com/stackrox/rox/central/detection/alertmanager",
				"github.com/stackrox/rox/central/detection/buildtime",
				"github.com/stackrox/rox/central/detection/lifecycle",
				"github.com/stackrox/rox/central/externalbackups/scheduler",
				"github.com/stackrox/rox/central/globaldb/v2backuprestore/manager",
				"github.com/stackrox/rox/central/graphql/generator/codegen",
				"github.com/stackrox/rox/central/graphql/handler",
				"github.com/stackrox/rox/central/graphql/resolvers",
				"github.com/stackrox/rox/central/graphql/resolvers/embeddedobjs",
				"github.com/stackrox/rox/central/graphql/resolvers/gen",
				"github.com/stackrox/rox/central/image/datastore",
				"github.com/stackrox/rox/central/image/datastore/store/common/v2",
				"github.com/stackrox/rox/central/image/datastore/store/postgres",
				"github.com/stackrox/rox/central/installation/store",
				"github.com/stackrox/rox/central/integrationhealth/reporter",
				"github.com/stackrox/rox/central/logimbue/handler",
				"github.com/stackrox/rox/central/metadata/service",
				"github.com/stackrox/rox/central/networkbaseline/manager",
				"github.com/stackrox/rox/central/networkgraph/aggregator",
				"github.com/stackrox/rox/central/networkgraph/entity/gatherer",
				"github.com/stackrox/rox/central/networkgraph/flow/datastore",
				"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store",
				"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/mocks",
				"github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store/postgres",
				"github.com/stackrox/rox/central/networkgraph/flow/datastore/mocks",
				"github.com/stackrox/rox/central/networkgraph/service",
				"github.com/stackrox/rox/central/networkpolicies/datastore",
				"github.com/stackrox/rox/central/networkpolicies/generator",
				"github.com/stackrox/rox/central/networkpolicies/service",
				"github.com/stackrox/rox/central/node/datastore",
				"github.com/stackrox/rox/central/node/datastore/store/common/v2",
				"github.com/stackrox/rox/central/node/datastore/store/postgres",
				"github.com/stackrox/rox/central/notifier/processor",
				"github.com/stackrox/rox/central/notifiers/awssh",
				"github.com/stackrox/rox/central/notifiers/cscc",
				"github.com/stackrox/rox/central/notifiers/jira",
				"github.com/stackrox/rox/central/notifiers/pagerduty",
				"github.com/stackrox/rox/central/notifiers/syslog",
				"github.com/stackrox/rox/central/pod/datastore",
				"github.com/stackrox/rox/central/postgres",
				"github.com/stackrox/rox/central/probeupload/manager",
				"github.com/stackrox/rox/central/processbaseline",
				"github.com/stackrox/rox/central/processbaseline/datastore",
				"github.com/stackrox/rox/central/processindicator/service",
				"github.com/stackrox/rox/central/pruning",
				"github.com/stackrox/rox/central/reports/config/datastore",
				"github.com/stackrox/rox/central/reports/scheduler",
				"github.com/stackrox/rox/central/reports/scheduler/v2",
				"github.com/stackrox/rox/central/reports/scheduler/v2/reportgenerator",
				"github.com/stackrox/rox/central/reprocessor",
				"github.com/stackrox/rox/central/scannerdefinitions/handler",
				"github.com/stackrox/rox/central/sensor/service",
				"github.com/stackrox/rox/central/sensor/service/connection",
				"github.com/stackrox/rox/central/sensor/service/connection/upgradecontroller",
				"github.com/stackrox/rox/central/sensor/service/pipeline/auditlogstateupdate",
				"github.com/stackrox/rox/central/sensor/service/pipeline/complianceoperatorresultsv2",
				"github.com/stackrox/rox/central/sensor/service/pipeline/imageintegrations",
				"github.com/stackrox/rox/central/sensor/service/pipeline/networkflowupdate",
				"github.com/stackrox/rox/central/sensor/service/pipeline/nodes",
				"github.com/stackrox/rox/central/sensor/telemetry",
				"github.com/stackrox/rox/central/sensorupgrade/controlservice",
				"github.com/stackrox/rox/central/splunk",
				"github.com/stackrox/rox/central/systeminfo/listener",
				"github.com/stackrox/rox/central/telemetry/gatherers",
				"github.com/stackrox/rox/central/version",
				"github.com/stackrox/rox/central/views/imagecve",
				"github.com/stackrox/rox/central/vulnerabilityrequest/datastore",
				"github.com/stackrox/rox/central/vulnerabilityrequest/manager/requestmgr",
				"github.com/stackrox/rox/central/vulnerabilityrequest/service",
				"github.com/stackrox/rox/central/vulnerabilityrequest/utils",
				"github.com/stackrox/rox/compliance/collection/auditlog",
				"github.com/stackrox/rox/compliance/collection/compliance",
				"github.com/stackrox/rox/compliance/collection/inventory",
				"github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_network_flows_partition/stores/previous",
				"github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_network_flows_partition/stores/updated",
				"github.com/stackrox/rox/migrator/migrations/m_175_to_m_176_create_notification_schedule_table",
				"github.com/stackrox/rox/migrator/migrations/m_180_to_m_181_move_to_blobstore",
				"github.com/stackrox/rox/migrator/migrations/m_182_to_m_183_remove_default_scope_manager_role",
				"github.com/stackrox/rox/migrator/migrations/m_186_to_m_187_add_blob_search",
				"github.com/stackrox/rox/migrator/migrations/m_95_to_m_96_alert_scoping_information_at_root",
				"github.com/stackrox/rox/migrator/migrations/n_04_to_n_05_postgres_images",
				"github.com/stackrox/rox/migrator/migrations/n_04_to_n_05_postgres_images/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_04_to_n_05_postgres_images/postgres",
				"github.com/stackrox/rox/migrator/migrations/n_27_to_n_28_postgres_log_imbues/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows",
				"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/postgres",
				"github.com/stackrox/rox/migrator/migrations/n_30_to_n_31_postgres_network_flows/store",
				"github.com/stackrox/rox/migrator/migrations/n_35_to_n_36_postgres_nodes/legacy",
				"github.com/stackrox/rox/migrator/migrations/n_35_to_n_36_postgres_nodes/postgres",
				"github.com/stackrox/rox/migrator/version",
				"github.com/stackrox/rox/pkg/administration/events",
				"github.com/stackrox/rox/pkg/auth/authproviders",
				"github.com/stackrox/rox/pkg/booleanpolicy",
				"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs",
				"github.com/stackrox/rox/pkg/booleanpolicy/evaluator",
				"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer",
				"github.com/stackrox/rox/pkg/clair",
				"github.com/stackrox/rox/pkg/compliance/checks/standards",
				"github.com/stackrox/rox/pkg/csv",
				"github.com/stackrox/rox/pkg/detection",
				"github.com/stackrox/rox/pkg/detection/deploytime",
				"github.com/stackrox/rox/pkg/detection/runtime",
				"github.com/stackrox/rox/pkg/fixtures",
				"github.com/stackrox/rox/pkg/grpc/common/authn/servicecerttoken",
				"github.com/stackrox/rox/pkg/images/enricher",
				"github.com/stackrox/rox/pkg/networkgraph/testutils",
				"github.com/stackrox/rox/pkg/notifiers",
				"github.com/stackrox/rox/pkg/postgres/pgutils",
				"github.com/stackrox/rox/pkg/postgres/walker",
				"github.com/stackrox/rox/pkg/readable",
				"github.com/stackrox/rox/pkg/registries/docker",
				"github.com/stackrox/rox/pkg/scanners/clairify",
				"github.com/stackrox/rox/pkg/scanners/clairv4",
				"github.com/stackrox/rox/pkg/scanners/google",
				"github.com/stackrox/rox/pkg/scanners/quay",
				"github.com/stackrox/rox/pkg/scanners/scannerv4",
				"github.com/stackrox/rox/pkg/scannerv4/client",
				"github.com/stackrox/rox/pkg/search/predicate",
				"github.com/stackrox/rox/pkg/search/predicate/basematchers",
				"github.com/stackrox/rox/pkg/telemetry",
				"github.com/stackrox/rox/pkg/timestamp",
				"github.com/stackrox/rox/pkg/timeutil",
				"github.com/stackrox/rox/scanner/mappers",
				"github.com/stackrox/rox/scanner/services",
				"github.com/stackrox/rox/sensor/admission-control/alerts",
				"github.com/stackrox/rox/sensor/admission-control/manager",
				"github.com/stackrox/rox/sensor/admission-control/settingswatch",
				"github.com/stackrox/rox/sensor/common/admissioncontroller",
				"github.com/stackrox/rox/sensor/common/clusterentities",
				"github.com/stackrox/rox/sensor/common/compliance",
				"github.com/stackrox/rox/sensor/common/detector",
				"github.com/stackrox/rox/sensor/common/detector/baseline",
				"github.com/stackrox/rox/sensor/common/networkflow/manager",
				"github.com/stackrox/rox/sensor/kubernetes/fake",
				"github.com/stackrox/rox/sensor/kubernetes/listener/resources",
				"github.com/stackrox/rox/sensor/kubernetes/listener/resources/complianceoperator/dispatchers",
				"github.com/stackrox/rox/sensor/kubernetes/telemetry",
				"github.com/stackrox/rox/tests",
				"github.com/stackrox/rox/tools/local-compliance",
			),
		},
		"github.com/magiconair/properties/assert": {
			replacement: "github.com/stretchr/testify/assert",
		},
		"github.com/prometheus/common/log": {
			replacement: "a logger",
		},
		"github.com/google/martian/log": {
			replacement: "a logger",
		},
		"github.com/gogo/protobuf/jsonpb": {
			replacement: "github.com/golang/protobuf/jsonpb",
		},
		"k8s.io/helm/...": {
			replacement: "package from helm.sh/v3",
		},
		"github.com/satori/go.uuid": {
			replacement: "github.com/stackrox/rox/pkg/uuid",
		},
		"github.com/google/uuid": {
			replacement: "github.com/stackrox/rox/pkg/uuid",
			allowlist: set.NewStringSet(
				"github.com/stackrox/rox/scanner/datastore/postgres/mocks", // Used by ClairCore.
			),
		},
	}
)

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "validateimports",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

type allowedPackage struct {
	path            string
	excludeChildren bool
}

func appendPackage(list []*allowedPackage, excludeChildren bool, pkgs ...string) []*allowedPackage {
	if list == nil {
		list = make([]*allowedPackage, len(pkgs))
	}

	for _, pkg := range pkgs {
		list = append(list, &allowedPackage{path: pkg, excludeChildren: excludeChildren})
	}
	return list
}

func appendPackageWithChildren(list []*allowedPackage, pkgs ...string) []*allowedPackage {
	return appendPackage(list, false, pkgs...)
}

func appendPackageWithoutChildren(list []*allowedPackage, pkgs ...string) []*allowedPackage {
	return appendPackage(list, true, pkgs...)
}

// Given the package name, get the root directory of the service.
// (The directory boundary that imports should not cross.)
func getRoot(packageName string) (root string, valid bool, err error) {
	if !strings.HasPrefix(packageName, roxPrefix) {
		return "", false, errors.Errorf("Package %s is not part of %s", packageName, roxPrefix)
	}
	unqualifiedPackageName := strings.TrimPrefix(packageName, roxPrefix)
	pathElems := strings.Split(unqualifiedPackageName, string(filepath.Separator))
	for i := len(pathElems); i > 0; i-- {
		validRoot := strings.Join(pathElems[:i], string(filepath.Separator))
		if validRoots.Contains(validRoot) {
			return validRoot, true, nil
		}
	}

	// We explicitly ignore directories with Go files that we don't want to
	// lint, and exit with an error if the package doesn't match any of these directories.
	// This will make sure that this target throws an error if someone
	// adds a new service.
	for _, ignoredRoot := range ignoredRoots {
		if strings.HasPrefix(unqualifiedPackageName, ignoredRoot) {
			return "", false, nil
		}
	}

	return "", false, errors.Errorf("Package %s not found in list. If you added a new build root, "+
		"you might need to add it to the validRoots list in tools/roxvet/analyzers/validateimports/analyzer.go.", packageName)
}

// verifySingleImportFromAllowedPackagesOnly returns true if the given import statement is allowed from the respective
// source package.
func verifySingleImportFromAllowedPackagesOnly(spec *ast.ImportSpec, packageName string, importRoot string, allowedPackages ...*allowedPackage) error {
	impPath, err := strconv.Unquote(spec.Path.Value)
	if err != nil {
		return err
	}

	if err := checkForbidden(impPath, packageName); err != nil {
		return err
	}

	if !strings.HasPrefix(impPath, roxPrefix) {
		return nil
	}

	trimmed := strings.TrimPrefix(impPath, roxPrefix)

	packagePaths := make([]string, 0, len(allowedPackages))
	for _, allowedPackage := range allowedPackages {
		if strings.HasPrefix(trimmed+"/", allowedPackage.path+"/") {
			if allowedPackage.excludeChildren && trimmed == allowedPackage.path {
				return nil
			}
			if !allowedPackage.excludeChildren {
				return nil
			}
		}
		packagePaths = append(packagePaths, allowedPackage.path)
	}
	return fmt.Errorf("%s cannot import from %s; only allowed roots are %+v", importRoot, trimmed, packagePaths)
}

// checkForbidden returns an error if an import has been forbidden and the importing package isn't in the allowlist
func checkForbidden(impPath, packageName string) error {
	forbiddenDetails, ok := forbiddenImports[impPath]
	for !ok {
		if !stringutils.ConsumeSuffix(&impPath, "/...") {
			impPath += "/..."
		} else {
			slashIdx := strings.LastIndex(impPath, "/")
			if slashIdx == -1 {
				return nil
			}
			impPath = impPath[:slashIdx] + "/..."
		}
		forbiddenDetails, ok = forbiddenImports[impPath]
	}

	if forbiddenDetails.replacement == packageName {
		return nil
	}

	if forbiddenDetails.allowlist.Contains(packageName) {
		return nil
	}

	return fmt.Errorf("import is illegal; use %q instead", forbiddenDetails.replacement)
}

// verifyImportsFromAllowedPackagesOnly verifies that all Go files in (subdirectories of) root
// only import StackRox code from allowedPackages
func verifyImportsFromAllowedPackagesOnly(pass *analysis.Pass, imports []*ast.ImportSpec, validImportRoot, packageName string) {
	allowedPackages := []*allowedPackage{{path: validImportRoot}, {path: "generated"}, {path: "image"}}
	// The migrator is NOT allowed to import all codes from pkg except isolated packages.
	if validImportRoot != "pkg" && !strings.HasPrefix(validImportRoot, "migrator") {
		allowedPackages = appendPackageWithChildren(allowedPackages, "pkg")
	}

	// Specific sub-packages in pkg that the migrator and its sub-packages are allowed to import go here.
	// Please be VERY prudent about adding to this list, since everything that's added to this list
	// will need to be protected by strict compatibility guarantees.
	// Keep this in alphabetic order.
	if strings.HasPrefix(validImportRoot, "migrator") {
		allowedPackages = appendPackageWithChildren(allowedPackages,
			"pkg/auth",
			"pkg/batcher",
			"pkg/binenc",
			"pkg/bolthelper",
			"pkg/booleanpolicy/policyversion",
			"pkg/buildinfo",
			"pkg/concurrency",
			"pkg/config",
			"pkg/cve",
			"pkg/cvss/cvssv2",
			"pkg/cvss/cvssv3",
			"pkg/dackbox",
			"pkg/dackbox/crud",
			"pkg/dackbox/raw",
			"pkg/dackbox/sortedkeys",
			"pkg/db",
			"pkg/dberrors",
			"pkg/dbhelper",
			"pkg/defaults/policies",
			"pkg/env",
			"pkg/errorhelpers",
			"pkg/features",
			"pkg/fileutils",
			"pkg/fsutils",
			"pkg/grpc/routes",
			"pkg/images/types",
			"pkg/ioutils",
			"pkg/jsonutil",
			"pkg/logging",
			"pkg/mathutil",
			"pkg/metrics",
			"pkg/migrations",
			"pkg/nodes/converter",
			"pkg/policyutils",
			"pkg/postgres/gorm",
			"pkg/postgres/pgadmin",
			"pkg/postgres/pgconfig",
			"pkg/postgres/pgtest",
			"pkg/postgres/pgutils",
			"pkg/postgres/walker",
			"pkg/probeupload",
			"pkg/process/normalize",
			"pkg/process/id",
			"pkg/protocompat",
			"pkg/protoconv",
			"pkg/retry",
			"pkg/rocksdb",
			"pkg/sac",
			"pkg/scancomponent",
			"pkg/scans",
			"pkg/search",
			"pkg/search/postgres",
			"pkg/secondarykey",
			"pkg/set",
			"pkg/sliceutils",
			"pkg/stringutils",
			"pkg/sync",
			"pkg/testutils",
			"pkg/timestamp",
			"pkg/utils",
			"pkg/uuid",
			"pkg/version",
		)

		allowedPackages = appendPackageWithoutChildren(allowedPackages, "pkg/postgres")

		// Migrations shall not depend on current schemas. Each migration can include a copy of the schema before and
		// after a specific migration.
		if validImportRoot == "migrator" {
			allowedPackages = appendPackageWithChildren(allowedPackages, "pkg/postgres/schema")
		}

		if validImportRoot == "migrator/migrations" {
			allowedPackages = appendPackageWithChildren(allowedPackages, "migrator")
		}
	}

	if validImportRoot == "sensor/debugger" {
		allowedPackages = appendPackageWithChildren(allowedPackages, "sensor/kubernetes/listener/resources", "sensor/kubernetes/client", "sensor/common/centralclient")
	}

	if validImportRoot == "tools" {
		allowedPackages = appendPackageWithChildren(allowedPackages,
			"central/globaldb", "central/metrics", "central/postgres", "pkg/sac/resources",
			"sensor/common/sensor", "sensor/common/centralclient", "sensor/kubernetes/client", "sensor/kubernetes/fake",
			"sensor/kubernetes/sensor", "sensor/debugger", "sensor/testutils",
			"compliance/collection/compliance", "compliance/collection/intervals")
	}

	if validImportRoot == "sensor/kubernetes" {
		allowedPackages = appendPackageWithChildren(allowedPackages, "sensor/common")
		allowedPackages = appendPackageWithChildren(allowedPackages, "sensor/utils")
	}

	// Allow scale tests to import some constants from central, to be more DRY.
	// This is not a problem since none of this code is used in prod anyway.
	if validImportRoot == "scale" {
		allowedPackages = appendPackageWithChildren(allowedPackages, "central")
	}

	if validImportRoot == "sensor/tests" {
		allowedPackages = appendPackageWithChildren(allowedPackages, "sensor/common", "sensor/kubernetes", "sensor/debugger", "sensor/testutils")
	}

	if validImportRoot == "sensor/common" {
		// Need this for unit tests.
		allowedPackages = appendPackageWithChildren(allowedPackages, "sensor/debugger")
	}

	for _, imp := range imports {
		err := verifySingleImportFromAllowedPackagesOnly(imp, packageName, validImportRoot, allowedPackages...)
		if err != nil {
			pass.Reportf(imp.Pos(), "invalid import %s: %v", imp.Path.Value, err)
		}
	}
}

func run(pass *analysis.Pass) (interface{}, error) {
	root, valid, err := getRoot(pass.Pkg.Path())
	if err != nil {
		pass.Reportf(token.NoPos, "couldn't find valid root: %v", err)
		return nil, nil
	}
	if !valid {
		return nil, nil
	}

	for _, file := range pass.Files {
		verifyImportsFromAllowedPackagesOnly(pass, file.Imports, root, pass.Pkg.Path())
	}

	return nil, nil
}
