export type ClusterType =
    | 'GENERIC_CLUSTER'
    | 'KUBERNETES_CLUSTER'
    | 'OPENSHIFT_CLUSTER'
    | 'OPENSHIFT4_CLUSTER';

export type ClusterLabels = Record<string, string>;

export type ClusterProviderMetadata =
    | ClusterGoogleProviderMetadata
    | ClusterAWSProviderMetadata
    | ClusterAzureProviderMetadata;

export type ClusterGoogleProviderMetadata = {
    google: GoogleProviderMetadata;
} & ClusterBaseProviderMetadata;

export type GoogleProviderMetadata = {
    project: string;
    clusterName: string;
};

export type ClusterAWSProviderMetadata = {
    aws: AWSProviderMetadata;
} & ClusterBaseProviderMetadata;

export type AWSProviderMetadata = {
    accountId: string;
};

export type ClusterAzureProviderMetadata = {
    azure: AzureProviderMetadata;
} & ClusterBaseProviderMetadata;

export type AzureProviderMetadata = {
    subscriptionId: string;
};

export type ClusterBaseProviderMetadata = {
    region: string;
    zone: string;
    verified: boolean;
};

export type ClusterOrchestratorMetadata = {
    version: string;
    openshiftVersion?: string;
    buildDate: string; // ISO 8601 date string
    apiVersions: string[];
};

export type CollectionMethod = 'UNSET_COLLECTION' | 'NO_COLLECTION' | 'EBPF' | 'CORE_BPF';

export type AdmissionControllerConfig = {
    enabled: boolean;
    timeoutSeconds: number; // int32
    scanInline: boolean;
    disableBypass: boolean;
    enforceOnUpdates: boolean;
};

export type TolerationsConfig = {
    disabled: boolean;
};

export type StaticClusterConfig = {
    type: ClusterType;
    mainImage: string;
    centralApiEndpoint: string;
    collectionMethod: CollectionMethod;
    collectorImage: string;
    admissionController: boolean;
    admissionControllerUpdates: boolean;
    tolerationsConfig: TolerationsConfig;
    slimCollector: boolean;
    admissionControllerEvents: boolean;
};

export type DynamicClusterConfig = {
    admissionControllerConfig: AdmissionControllerConfig;
    registryOverride: string;
    disableAuditLogs: boolean;
};

// Encodes a complete cluster configuration minus ID/Name identifiers
// including static and dynamic settings.
export type CompleteClusterConfig = {
    dynamicConfig: DynamicClusterConfig;
    staticConfig: StaticClusterConfig;
    configFingerprint: string;
    clusterLabels: Record<string, string>;
};

// SensorDeploymentIdentification aims at uniquely identifying a Sensor deployment. It is used to determine
// whether a sensor connection comes from a sensor pod that has restarted or was recreated (possibly after a network
// partition), or from a deployment in a different namespace or cluster.
export type SensorDeploymentIdentification = {
    systemNamespaceId: string;
    defaultNamespaceId: string;
    appNamespace: string;
    appNamespaceId: string;
    appServiceaccountId: string;
    k8sNodeName: string;
};

export type Cluster = {
    id: string;
    name: string;
    type: ClusterType;
    labels: ClusterLabels;
    mainImage: string;
    collectorImage: string;
    centralApiEndpoint: string;
    runtimeSupport: boolean; // deprecated
    collectionMethod: CollectionMethod;
    admissionController: boolean;
    admissionControllerUpdates: boolean;
    admissionControllerEvents: boolean;
    status: ClusterStatus;
    dynamicConfig: DynamicClusterConfig;
    tolerationsConfig: TolerationsConfig;
    priority: string; // int64
    healthStatus: ClusterHealthStatus;
    slimCollector: boolean;

    // The Helm configuration of a cluster is only present in case the cluster is Helm- or Operator-managed.
    helmConfig: CompleteClusterConfig;

    // most_recent_sensor_id is the current or most recent identification of a successfully connected sensor (if any).
    mostRecentSensorId: SensorDeploymentIdentification;

    // For internal use only.
    auditLogState: Record<string, AuditLogFileState>;

    initBundleId: string;
    managedBy: ClusterManagerType;
};

export type ClusterManagerType =
    | 'MANAGER_TYPE_UNKNOWN'
    | 'MANAGER_TYPE_MANUAL'
    | 'MANAGER_TYPE_HELM_CHART'
    | 'MANAGER_TYPE_KUBERNETES_OPERATOR';

export type ClusterCertExpiryStatus = {
    sensorCertExpiry: string; // ISO 8601 date string
};

export type ClusterStatus = {
    sensorVersion: string;
    // DEPRECATED_last_contact
    providerMetadata: ClusterProviderMetadata;
    orchestratorMetadata: ClusterOrchestratorMetadata;
    upgradeStatus: ClusterUpgradeStatus;
    certExpiryStatus: ClusterCertExpiryStatus;
};

export type ClusterUpgradability =
    | 'UNSET'
    | 'UP_TO_DATE'
    | 'MANUAL_UPGRADE_REQUIRED'
    | 'AUTO_UPGRADE_POSSIBLE'
    | 'SENSOR_VERSION_HIGHER';
// SENSOR_VERSION_HIGHER occurs when we detect that the sensor
// is running a newer version than this Central. This is unexpected,
// but can occur depending on the patches a customer does.
// In this case, we will NOT automatically "upgrade" the sensor,
// since that would be a downgrade, even if the autoupgrade setting is
// on. The user will be allowed to manually trigger the upgrade, but they are
// strongly discouraged from doing so without upgrading Central first, since this
// is an unsupported configuration.

export type ClusterUpgradeStatus = {
    upgradability: ClusterUpgradability;
    upgradabilityStatusReason: string;

    // The progress of the current or most recent upgrade, if any,
    // Note that we don't store any historical data -- the moment
    // a new upgrade attempt is triggered, we overwrite
    // information from the previous attempt.
    mostRecentProcess: UpgradeProcessStatus;
};

export type ClusterUpgradeProcessType = 'UPGRADE' | 'CERT_ROTATION';
// UPGRADE represents a sensor version upgrade.
// CERT_ROTATION represents an upgrade process that only rotates the TLS certs
// used by the cluster, without changing anything else.

export type UpgradeProcessStatus = {
    active: boolean;
    id: string;
    targetVersion: string; // only relevant if type == Upgrade
    upgraderImage: string;
    initiatedAt: string; // ISO 8601 date string
    progress: ClusterUpgradeProgress;
    type: ClusterUpgradeProcessType;
};

export type ClusterUpgradeState =
    | 'UPGRADE_INITIALIZING'

    // In-progress states.
    | 'UPGRADER_LAUNCHING'
    | 'UPGRADER_LAUNCHED'
    | 'PRE_FLIGHT_CHECKS_COMPLETE'
    | 'UPGRADE_OPERATIONS_DONE'

    // The success state.
    // PLEASE NUMBER ALL IN-PROGRESS STATES ABOVE THIS
    // AND ALL ERROR STATES BELOW THIS.
    | 'UPGRADE_COMPLETE'

    // Error states.
    | 'UPGRADE_INITIALIZATION_ERROR'
    | 'PRE_FLIGHT_CHECKS_FAILED'
    | 'UPGRADE_ERROR_ROLLING_BACK'
    | 'UPGRADE_ERROR_ROLLED_BACK'
    | 'UPGRADE_ERROR_ROLLBACK_FAILED'
    | 'UPGRADE_ERROR_UNKNOWN'
    | 'UPGRADE_TIMED_OUT';

export type ClusterUpgradeProgress = {
    upgradeState: ClusterUpgradeState;
    upgradeStatusDetail: string;
    since: string; // ISO 8601 date string
};

export type ClusterCVEEdge = {
    id: string; // base 64 encoded Cluster:CVE ids
    isFixable: boolean;
    fixedBy?: string; // Whether there is a version the CVE is fixed in the Cluster
};

// AuditLogFileState tracks the last audit log event timestamp and ID that was collected by Compliance
// For internal use only
export type AuditLogFileState = {
    collectLogsSince: string; // ISO 8601 date string
    lastAuditId: string; // Previously received audit id. May be empty
};

export type ClusterHealthStatusLabel =
    | 'UNINITIALIZED'
    | 'UNAVAILABLE' // Only collector can have unavailable status
    | 'UNHEALTHY'
    | 'DEGRADED'
    | 'HEALTHY';

export type ClusterHealthStatus = {
    collectorHealthInfo: CollectorHealthInfo;
    admissionControlHealthInfo: AdmissionControlHealthInfo;
    sensorHealthStatus: ClusterHealthStatusLabel;
    collectorHealthStatus: ClusterHealthStatusLabel;
    overallHealthStatus: ClusterHealthStatusLabel;
    admissionControlHealthStatus: ClusterHealthStatusLabel;

    // For sensors not having health capability, this will be filled with gRPC connection poll. Otherwise,
    // this timestamp will be updated by central pipeline when message is processed
    lastContact: string; // ISO 8601 date string

    // To track cases such as when sensor is healthy, but collector status data is unavailable because the sensor is on an old version
    healthInfoComplete: boolean;
};

// CollectorHealthInfo carries data about collector deployment but does not include collector health status derived from this data.
// Aggregated collector health status is not included because it is derived in central and not in the component that
// first reports CollectorHealthInfo (sensor).
export type CollectorHealthInfo = {
    // This is the version of the collector deamonset as returned by k8s API
    version: string;

    // The following fields are made optional/nullable because there can be errors when trying to obtain them and
    // the default value of 0 might be confusing with the actual value 0. In case an error happens when trying to obtain
    // a certain field, it will be absent (instead of having the default value).
    totalDesiredPods?: number; // int32
    totalReadyPods?: number; // int32
    totalRegisteredNodes?: number; // int32

    // Collection of errors that occurred while trying to obtain collector health info.
    statusErrors: string[];
};

// AdmissionControlHealthInfo carries data about admission control deployment but does not include admission control health status
// derived from this data.
// Aggregated admission control health status is not included because it is derived in central and not in the component that
// first reports AdmissionControlHealthInfo (sensor).
export type AdmissionControlHealthInfo = {
    // The following fields are made optional/nullable because there can be errors when trying to obtain them and
    // the default value of 0 might be confusing with the actual value 0. In case an error happens when trying to obtain
    // a certain field, it will be absent (instead of having the default value).
    totalDesiredPods?: number; // int32
    totalReadyPods?: number; // int32

    // Collection of errors that occurred while trying to obtain admission control health info.
    statusErrors: string[];
};
