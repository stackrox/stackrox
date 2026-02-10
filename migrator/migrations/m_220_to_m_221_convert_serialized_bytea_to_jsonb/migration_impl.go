package m220tom221

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var (
	log       = loghelper.LogWrapper{}
	batchSize = 5000

	// tableToProtoName maps each SQL table name to its protobuf full name.
	// Only top-level tables have a serialized column; child tables do not.
	tableToProtoName = map[string]protoreflect.FullName{
		"administration_events":                       "storage.AdministrationEvent",
		"alerts":                                      "storage.Alert",
		"api_tokens":                                  "storage.TokenMetadata",
		"auth_machine_to_machine_configs":              "storage.AuthMachineToMachineConfig",
		"auth_providers":                              "storage.AuthProvider",
		"base_images":                                 "storage.BaseImage",
		"base_image_tags":                             "storage.BaseImageTag",
		"base_image_layers":                           "storage.BaseImageLayer",
		"base_image_repositories":                     "storage.BaseImageRepository",
		"blobs":                                       "storage.Blob",
		"cloud_sources":                               "storage.CloudSource",
		"cluster_cve_edges":                           "storage.ClusterCVEEdge",
		"cluster_cves":                                "storage.ClusterCVE",
		"cluster_health_statuses":                     "storage.ClusterHealthStatus",
		"cluster_init_bundles":                        "storage.InitBundleMeta",
		"clusters":                                    "storage.Cluster",
		"collections":                                 "storage.ResourceCollection",
		"compliance_configs":                          "storage.ComplianceConfig",
		"compliance_domains":                          "storage.ComplianceDomain",
		"compliance_integrations":                     "storage.ComplianceIntegration",
		"compliance_operator_check_result_v2":         "storage.ComplianceOperatorCheckResultV2",
		"compliance_operator_check_results":           "storage.ComplianceOperatorCheckResult",
		"compliance_operator_cluster_scan_config_statuses": "storage.ComplianceOperatorClusterScanConfigStatus",
		"compliance_operator_profile_v2":              "storage.ComplianceOperatorProfileV2",
		"compliance_operator_profiles":                "storage.ComplianceOperatorProfile",
		"compliance_operator_remediation_v2":          "storage.ComplianceOperatorRemediationV2",
		"compliance_operator_report_snapshot_v2":      "storage.ComplianceOperatorReportSnapshotV2",
		"compliance_operator_rule_v2":                 "storage.ComplianceOperatorRuleV2",
		"compliance_operator_rules":                   "storage.ComplianceOperatorRule",
		"compliance_operator_scan_configuration_v2":   "storage.ComplianceOperatorScanConfigurationV2",
		"compliance_operator_scan_setting_binding_v2": "storage.ComplianceOperatorScanSettingBindingV2",
		"compliance_operator_scan_setting_bindings":   "storage.ComplianceOperatorScanSettingBinding",
		"compliance_operator_scan_v2":                 "storage.ComplianceOperatorScanV2",
		"compliance_operator_scans":                   "storage.ComplianceOperatorScan",
		"compliance_operator_suite_v2":                "storage.ComplianceOperatorSuiteV2",
		"compliance_run_metadata":                     "storage.ComplianceRunMetadata",
		"compliance_run_results":                      "storage.ComplianceRunResults",
		"compliance_strings":                          "storage.ComplianceStrings",
		"configs":                                     "storage.Config",
		"declarative_config_healths":                  "storage.DeclarativeConfigHealth",
		"delegated_registry_configs":                  "storage.DelegatedRegistryConfig",
		"deployments":                                 "storage.Deployment",
		"discovered_clusters":                         "storage.DiscoveredCluster",
		"external_backups":                            "storage.ExternalBackup",
		"groups":                                      "storage.Group",
		"hashes":                                      "storage.Hash",
		"image_component_v2":                          "storage.ImageComponentV2",
		"image_cve_infos":                             "storage.ImageCVEInfo",
		"image_cves_v2":                               "storage.ImageCVEV2",
		"image_integrations":                          "storage.ImageIntegration",
		"images":                                      "storage.Image",
		"images_v2":                                   "storage.Image",
		"installation_infos":                          "storage.InstallationInfo",
		"integration_healths":                         "storage.IntegrationHealth",
		"k8s_roles":                                   "storage.K8SRole",
		"listening_endpoints":                         "storage.ProcessListeningOnPortStorage",
		"log_imbues":                                  "storage.LogImbue",
		"namespaces":                                  "storage.NamespaceMetadata",
		"network_baselines":                           "storage.NetworkBaseline",
		"network_entities":                            "storage.NetworkEntity",
		"network_graph_configs":                       "storage.NetworkGraphConfig",
		"networkpolicies":                             "storage.NetworkPolicy",
		"networkpoliciesundodeployments":               "storage.NetworkPolicyApplicationUndoDeploymentRecord",
		"networkpolicyapplicationundorecords":          "storage.NetworkPolicyApplicationUndoRecord",
		"node_component_edges":                        "storage.NodeComponentEdge",
		"node_components":                             "storage.NodeComponent",
		"node_components_cves_edges":                  "storage.NodeComponentCVEEdge",
		"node_cves":                                   "storage.NodeCVE",
		"nodes":                                       "storage.Node",
		"notification_schedules":                      "storage.NotificationSchedule",
		"notifier_enc_configs":                        "storage.NotifierEncConfig",
		"notifiers":                                   "storage.Notifier",
		"permission_sets":                             "storage.PermissionSet",
		"pods":                                        "storage.Pod",
		"policies":                                    "storage.Policy",
		"policy_categories":                           "storage.PolicyCategory",
		"policy_category_edges":                       "storage.PolicyCategoryEdge",
		"process_baseline_results":                    "storage.ProcessBaselineResults",
		"process_baselines":                           "storage.ProcessBaseline",
		"process_indicators":                          "storage.ProcessIndicator",
		"report_configurations":                       "storage.ReportConfiguration",
		"report_snapshots":                            "storage.ReportSnapshot",
		"risks":                                       "storage.Risk",
		"role_bindings":                               "storage.K8SRoleBinding",
		"roles":                                       "storage.Role",
		"secrets":                                     "storage.Secret",
		"secured_units":                               "storage.SecuredUnits",
		"sensor_upgrade_configs":                      "storage.SensorUpgradeConfig",
		"service_accounts":                            "storage.ServiceAccount",
		"service_identities":                          "storage.ServiceIdentity",
		"signature_integrations":                      "storage.SignatureIntegration",
		"simple_access_scopes":                        "storage.SimpleAccessScope",
		"system_infos":                                "storage.SystemInfo",
		"versions":                                    "storage.Version",
		"virtual_machines":                            "storage.VirtualMachine",
		"vulnerability_requests":                      "storage.VulnerabilityRequest",
		"watched_images":                              "storage.WatchedImage",
	}
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())

	for tableName, protoName := range tableToProtoName {
		if err := convertTable(ctx, database.PostgresDB, tableName, protoName); err != nil {
			log.WriteToStderrf("failed to convert table %s: %v", tableName, err)
			return errors.Wrapf(err, "converting table %s", tableName)
		}
	}

	return nil
}

func convertTable(ctx context.Context, db postgres.DB, tableName string, protoName protoreflect.FullName) error {
	// Check if the table exists and has a serialized column of type bytea.
	var colType string
	err := db.QueryRow(ctx,
		`SELECT data_type FROM information_schema.columns WHERE table_name = $1 AND column_name = 'serialized'`,
		tableName,
	).Scan(&colType)
	if err != nil {
		// Table doesn't exist or has no serialized column — skip.
		log.WriteToStderrf("skipping table %s: %v", tableName, err)
		return nil
	}
	if colType == "jsonb" {
		// Already converted — skip.
		return nil
	}

	msgType, err := protoregistry.GlobalTypes.FindMessageByName(protoName)
	if err != nil {
		return errors.Wrapf(err, "proto type %s not found in registry", protoName)
	}

	// Get the primary key column(s) for the table.
	pkCol, err := getPrimaryKeyColumn(ctx, db, tableName)
	if err != nil {
		return errors.Wrapf(err, "getting primary key for table %s", tableName)
	}

	// Process in batches using a cursor-like approach.
	offset := 0
	totalConverted := 0
	for {
		query := fmt.Sprintf(
			`SELECT %s, serialized FROM %s ORDER BY %s LIMIT %d OFFSET %d`,
			pkCol, tableName, pkCol, batchSize, offset,
		)
		rows, err := db.Query(ctx, query)
		if err != nil {
			return errors.Wrapf(err, "querying table %s", tableName)
		}

		type rowData struct {
			pk   interface{}
			json []byte
		}
		var batch []rowData
		for rows.Next() {
			var pk interface{}
			var serialized []byte
			if err := rows.Scan(&pk, &serialized); err != nil {
				rows.Close()
				return errors.Wrapf(err, "scanning row from %s", tableName)
			}
			if serialized == nil {
				continue
			}

			msg := msgType.New().Interface()
			if err := proto.Unmarshal(serialized, msg); err != nil {
				rows.Close()
				return errors.Wrapf(err, "unmarshaling protobuf for table %s", tableName)
			}

			jsonBytes, err := protojson.Marshal(msg)
			if err != nil {
				rows.Close()
				return errors.Wrapf(err, "marshaling to JSON for table %s", tableName)
			}

			batch = append(batch, rowData{pk: pk, json: jsonBytes})
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return errors.Wrapf(err, "iterating rows from %s", tableName)
		}

		if len(batch) == 0 {
			break
		}

		// Update the rows with JSON data.
		for _, rd := range batch {
			updateQuery := fmt.Sprintf(
				`UPDATE %s SET serialized = $1 WHERE %s = $2`,
				tableName, pkCol,
			)
			if _, err := db.Exec(ctx, updateQuery, rd.json, rd.pk); err != nil {
				return errors.Wrapf(err, "updating row in %s", tableName)
			}
		}

		totalConverted += len(batch)
		offset += len(batch)

		if len(batch) < batchSize {
			break
		}
	}

	// ALTER COLUMN type from bytea to jsonb.
	alterStmt := fmt.Sprintf(
		`ALTER TABLE %s ALTER COLUMN serialized TYPE jsonb USING convert_from(serialized, 'UTF-8')::jsonb`,
		tableName,
	)
	if _, err := db.Exec(ctx, alterStmt); err != nil {
		return errors.Wrapf(err, "altering column type for table %s", tableName)
	}

	log.WriteToStderrf("converted %d rows in table %s from bytea to jsonb", totalConverted, tableName)
	return nil
}

func getPrimaryKeyColumn(ctx context.Context, db postgres.DB, tableName string) (string, error) {
	var pkCol string
	err := db.QueryRow(ctx, `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		WHERE i.indrelid = $1::regclass AND i.indisprimary
		ORDER BY a.attnum
		LIMIT 1
	`, tableName).Scan(&pkCol)
	if err != nil {
		return "", err
	}
	return pkCol, nil
}
