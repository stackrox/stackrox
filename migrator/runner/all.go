package runner

import (
	// Import these packages to trigger the registration.
	_ "github.com/stackrox/rox/migrator/migrations/m_16_to_m_17_add_scan_stats"
	_ "github.com/stackrox/rox/migrator/migrations/m_17_to_m_18_alert_retention_config"
	_ "github.com/stackrox/rox/migrator/migrations/m_18_to_m_19_badger_migration"
	_ "github.com/stackrox/rox/migrator/migrations/m_19_to_m_20_badgerdb_duplication"
	_ "github.com/stackrox/rox/migrator/migrations/m_20_to_m_21_group_colon_migration"
	_ "github.com/stackrox/rox/migrator/migrations/m_21_to_m_22_default_tolerations_disabled"
	_ "github.com/stackrox/rox/migrator/migrations/m_22_to_m_23_delete_cis_docker_1_1_0_cis_k8s_1_2_0"
	_ "github.com/stackrox/rox/migrator/migrations/m_23_to_m_24_delete_cis_k8s_1_4_1"
	_ "github.com/stackrox/rox/migrator/migrations/m_24_to_m_25_update_policy_texts"
	_ "github.com/stackrox/rox/migrator/migrations/m_25_to_m_26_starttls_auth"
	_ "github.com/stackrox/rox/migrator/migrations/m_26_to_m_27_prune_orphaned_process_whitelists"
	_ "github.com/stackrox/rox/migrator/migrations/m_27_to_m_28_dackbox"
	_ "github.com/stackrox/rox/migrator/migrations/m_28_to_m_29_img_scan_stats"
	_ "github.com/stackrox/rox/migrator/migrations/m_29_to_m_30_alert_deployment_ns_id"
	_ "github.com/stackrox/rox/migrator/migrations/m_30_to_m_31_group_key_format"
	_ "github.com/stackrox/rox/migrator/migrations/m_31_to_m_32_remove_unique_indicators"
	_ "github.com/stackrox/rox/migrator/migrations/m_32_to_m_33_dackbox"
	_ "github.com/stackrox/rox/migrator/migrations/m_33_to_m_34_graph_schema"
	_ "github.com/stackrox/rox/migrator/migrations/m_34_to_m_35_apitoken_multiple_roles"
	_ "github.com/stackrox/rox/migrator/migrations/m_35_to_m_36_normalize_clusters"
	_ "github.com/stackrox/rox/migrator/migrations/m_36_to_m_37_rocksdb"
)
