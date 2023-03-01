/* eslint-disable */
import { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
    ID: string;
    String: string;
    Boolean: boolean;
    Int: number;
    Float: number;
    Time: string;
};

export type AwsProviderMetadata = {
    __typename?: 'AWSProviderMetadata';
    accountId: Scalars['String'];
};

export type AwsSecurityHub = {
    __typename?: 'AWSSecurityHub';
    accountId: Scalars['String'];
    credentials?: Maybe<AwsSecurityHub_Credentials>;
    region: Scalars['String'];
};

export type AwsSecurityHub_Credentials = {
    __typename?: 'AWSSecurityHub_Credentials';
    accessKeyId: Scalars['String'];
    secretAccessKey: Scalars['String'];
};

export enum Access {
    NoAccess = 'NO_ACCESS',
    ReadAccess = 'READ_ACCESS',
    ReadWriteAccess = 'READ_WRITE_ACCESS',
}

export type ActiveComponent_ActiveContext = {
    __typename?: 'ActiveComponent_ActiveContext';
    containerName: Scalars['String'];
    imageId: Scalars['String'];
};

export type ActiveState = {
    __typename?: 'ActiveState';
    activeContexts: Array<ActiveComponent_ActiveContext>;
    state: Scalars['String'];
};

export type AdmissionControlHealthInfo = {
    __typename?: 'AdmissionControlHealthInfo';
    statusErrors: Array<Scalars['String']>;
};

export type AdmissionControllerConfig = {
    __typename?: 'AdmissionControllerConfig';
    disableBypass: Scalars['Boolean'];
    enabled: Scalars['Boolean'];
    enforceOnUpdates: Scalars['Boolean'];
    scanInline: Scalars['Boolean'];
    timeoutSeconds: Scalars['Int'];
};

export type AggregateBy = {
    aggregateFunc?: InputMaybe<Scalars['String']>;
    distinct?: InputMaybe<Scalars['Boolean']>;
};

export type Alert = {
    __typename?: 'Alert';
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    deployment?: Maybe<Alert_Deployment>;
    enforcement?: Maybe<Alert_Enforcement>;
    entity?: Maybe<AlertEntity>;
    firstOccurred?: Maybe<Scalars['Time']>;
    id: Scalars['ID'];
    image?: Maybe<ContainerImage>;
    lifecycleStage: LifecycleStage;
    namespace: Scalars['String'];
    namespaceId: Scalars['String'];
    policy?: Maybe<Policy>;
    processViolation?: Maybe<Alert_ProcessViolation>;
    resolvedAt?: Maybe<Scalars['Time']>;
    resource?: Maybe<Alert_Resource>;
    snoozeTill?: Maybe<Scalars['Time']>;
    state: ViolationState;
    time?: Maybe<Scalars['Time']>;
    unusedVarSink?: Maybe<Scalars['Int']>;
    violations: Array<Maybe<Alert_Violation>>;
};

export type AlertUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type AlertEntity = Alert_Deployment | Alert_Resource | ContainerImage;

export type Alert_Deployment = {
    __typename?: 'Alert_Deployment';
    annotations: Array<Label>;
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    containers: Array<Maybe<Alert_Deployment_Container>>;
    id: Scalars['ID'];
    inactive: Scalars['Boolean'];
    labels: Array<Label>;
    name: Scalars['String'];
    namespace: Scalars['String'];
    namespaceId: Scalars['String'];
    type: Scalars['String'];
};

export type Alert_Deployment_Container = {
    __typename?: 'Alert_Deployment_Container';
    image?: Maybe<ContainerImage>;
    name: Scalars['String'];
};

export type Alert_Enforcement = {
    __typename?: 'Alert_Enforcement';
    action: EnforcementAction;
    message: Scalars['String'];
};

export type Alert_ProcessViolation = {
    __typename?: 'Alert_ProcessViolation';
    message: Scalars['String'];
    processes: Array<Maybe<ProcessIndicator>>;
};

export type Alert_Resource = {
    __typename?: 'Alert_Resource';
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    name: Scalars['String'];
    namespace: Scalars['String'];
    namespaceId: Scalars['String'];
    resourceType: Alert_Resource_ResourceType;
};

export enum Alert_Resource_ResourceType {
    Configmaps = 'CONFIGMAPS',
    Secrets = 'SECRETS',
    Unknown = 'UNKNOWN',
}

export type Alert_Violation = {
    __typename?: 'Alert_Violation';
    keyValueAttrs?: Maybe<Alert_Violation_KeyValueAttrs>;
    message: Scalars['String'];
    messageAttributes?: Maybe<Alert_ViolationMessageAttributes>;
    networkFlowInfo?: Maybe<Alert_Violation_NetworkFlowInfo>;
    time?: Maybe<Scalars['Time']>;
    type: Alert_Violation_Type;
};

export type Alert_ViolationMessageAttributes =
    | Alert_Violation_KeyValueAttrs
    | Alert_Violation_NetworkFlowInfo;

export type Alert_Violation_KeyValueAttrs = {
    __typename?: 'Alert_Violation_KeyValueAttrs';
    attrs: Array<Maybe<Alert_Violation_KeyValueAttrs_KeyValueAttr>>;
};

export type Alert_Violation_KeyValueAttrs_KeyValueAttr = {
    __typename?: 'Alert_Violation_KeyValueAttrs_KeyValueAttr';
    key: Scalars['String'];
    value: Scalars['String'];
};

export type Alert_Violation_NetworkFlowInfo = {
    __typename?: 'Alert_Violation_NetworkFlowInfo';
    destination?: Maybe<Alert_Violation_NetworkFlowInfo_Entity>;
    protocol: L4Protocol;
    source?: Maybe<Alert_Violation_NetworkFlowInfo_Entity>;
};

export type Alert_Violation_NetworkFlowInfo_Entity = {
    __typename?: 'Alert_Violation_NetworkFlowInfo_Entity';
    deploymentNamespace: Scalars['String'];
    deploymentType: Scalars['String'];
    entityType: NetworkEntityInfo_Type;
    name: Scalars['String'];
    port: Scalars['Int'];
};

export enum Alert_Violation_Type {
    Generic = 'GENERIC',
    K8SEvent = 'K8S_EVENT',
    NetworkFlow = 'NETWORK_FLOW',
    NetworkPolicy = 'NETWORK_POLICY',
}

export type AzureProviderMetadata = {
    __typename?: 'AzureProviderMetadata';
    subscriptionId: Scalars['String'];
};

export enum BooleanOperator {
    And = 'AND',
    Or = 'OR',
}

export type Cscc = {
    __typename?: 'CSCC';
    serviceAccount: Scalars['String'];
    sourceId: Scalars['String'];
};

export type Cve = {
    __typename?: 'CVE';
    createdAt?: Maybe<Scalars['Time']>;
    id: Scalars['ID'];
    impactScore: Scalars['Float'];
    lastModified?: Maybe<Scalars['Time']>;
    link: Scalars['String'];
    publishedOn?: Maybe<Scalars['Time']>;
    references: Array<Maybe<Cve_Reference>>;
    scoreVersion: Cve_ScoreVersion;
    severity: VulnerabilitySeverity;
    summary: Scalars['String'];
    suppressActivation?: Maybe<Scalars['Time']>;
    suppressExpiry?: Maybe<Scalars['Time']>;
    suppressed: Scalars['Boolean'];
    type: Cve_CveType;
    types: Array<Cve_CveType>;
};

export type CveInfo = {
    __typename?: 'CVEInfo';
    createdAt?: Maybe<Scalars['Time']>;
    cve: Scalars['String'];
    cvssV2?: Maybe<Cvssv2>;
    cvssV3?: Maybe<Cvssv3>;
    lastModified?: Maybe<Scalars['Time']>;
    link: Scalars['String'];
    publishedOn?: Maybe<Scalars['Time']>;
    references: Array<Maybe<CveInfo_Reference>>;
    scoreVersion: CveInfo_ScoreVersion;
    summary: Scalars['String'];
};

export type CveInfo_Reference = {
    __typename?: 'CVEInfo_Reference';
    tags: Array<Scalars['String']>;
    uRI: Scalars['String'];
};

export enum CveInfo_ScoreVersion {
    Unknown = 'UNKNOWN',
    V2 = 'V2',
    V3 = 'V3',
}

export enum Cve_CveType {
    ImageCve = 'IMAGE_CVE',
    IstioCve = 'ISTIO_CVE',
    K8SCve = 'K8S_CVE',
    NodeCve = 'NODE_CVE',
    OpenshiftCve = 'OPENSHIFT_CVE',
    UnknownCve = 'UNKNOWN_CVE',
}

export type Cve_Reference = {
    __typename?: 'CVE_Reference';
    tags: Array<Scalars['String']>;
    uRI: Scalars['String'];
};

export enum Cve_ScoreVersion {
    Unknown = 'UNKNOWN',
    V2 = 'V2',
    V3 = 'V3',
}

export type Cvssv2 = {
    __typename?: 'CVSSV2';
    accessComplexity: Cvssv2_AccessComplexity;
    attackVector: Cvssv2_AttackVector;
    authentication: Cvssv2_Authentication;
    availability: Cvssv2_Impact;
    confidentiality: Cvssv2_Impact;
    exploitabilityScore: Scalars['Float'];
    impactScore: Scalars['Float'];
    integrity: Cvssv2_Impact;
    score: Scalars['Float'];
    severity: Cvssv2_Severity;
    vector: Scalars['String'];
};

export enum Cvssv2_AccessComplexity {
    AccessHigh = 'ACCESS_HIGH',
    AccessLow = 'ACCESS_LOW',
    AccessMedium = 'ACCESS_MEDIUM',
}

export enum Cvssv2_AttackVector {
    AttackAdjacent = 'ATTACK_ADJACENT',
    AttackLocal = 'ATTACK_LOCAL',
    AttackNetwork = 'ATTACK_NETWORK',
}

export enum Cvssv2_Authentication {
    AuthMultiple = 'AUTH_MULTIPLE',
    AuthNone = 'AUTH_NONE',
    AuthSingle = 'AUTH_SINGLE',
}

export enum Cvssv2_Impact {
    ImpactComplete = 'IMPACT_COMPLETE',
    ImpactNone = 'IMPACT_NONE',
    ImpactPartial = 'IMPACT_PARTIAL',
}

export enum Cvssv2_Severity {
    High = 'HIGH',
    Low = 'LOW',
    Medium = 'MEDIUM',
    Unknown = 'UNKNOWN',
}

export type Cvssv3 = {
    __typename?: 'CVSSV3';
    attackComplexity: Cvssv3_Complexity;
    attackVector: Cvssv3_AttackVector;
    availability: Cvssv3_Impact;
    confidentiality: Cvssv3_Impact;
    exploitabilityScore: Scalars['Float'];
    impactScore: Scalars['Float'];
    integrity: Cvssv3_Impact;
    privilegesRequired: Cvssv3_Privileges;
    scope: Cvssv3_Scope;
    score: Scalars['Float'];
    severity: Cvssv3_Severity;
    userInteraction: Cvssv3_UserInteraction;
    vector: Scalars['String'];
};

export enum Cvssv3_AttackVector {
    AttackAdjacent = 'ATTACK_ADJACENT',
    AttackLocal = 'ATTACK_LOCAL',
    AttackNetwork = 'ATTACK_NETWORK',
    AttackPhysical = 'ATTACK_PHYSICAL',
}

export enum Cvssv3_Complexity {
    ComplexityHigh = 'COMPLEXITY_HIGH',
    ComplexityLow = 'COMPLEXITY_LOW',
}

export enum Cvssv3_Impact {
    ImpactHigh = 'IMPACT_HIGH',
    ImpactLow = 'IMPACT_LOW',
    ImpactNone = 'IMPACT_NONE',
}

export enum Cvssv3_Privileges {
    PrivilegeHigh = 'PRIVILEGE_HIGH',
    PrivilegeLow = 'PRIVILEGE_LOW',
    PrivilegeNone = 'PRIVILEGE_NONE',
}

export enum Cvssv3_Scope {
    Changed = 'CHANGED',
    Unchanged = 'UNCHANGED',
}

export enum Cvssv3_Severity {
    Critical = 'CRITICAL',
    High = 'HIGH',
    Low = 'LOW',
    Medium = 'MEDIUM',
    None = 'NONE',
    Unknown = 'UNKNOWN',
}

export enum Cvssv3_UserInteraction {
    UiNone = 'UI_NONE',
    UiRequired = 'UI_REQUIRED',
}

export type Cert = {
    __typename?: 'Cert';
    algorithm: Scalars['String'];
    endDate?: Maybe<Scalars['Time']>;
    issuer?: Maybe<CertName>;
    sans: Array<Scalars['String']>;
    startDate?: Maybe<Scalars['Time']>;
    subject?: Maybe<CertName>;
};

export type CertName = {
    __typename?: 'CertName';
    commonName: Scalars['String'];
    country: Scalars['String'];
    locality: Scalars['String'];
    names: Array<Scalars['String']>;
    organization: Scalars['String'];
    organizationUnit: Scalars['String'];
    postalCode: Scalars['String'];
    province: Scalars['String'];
    streetAddress: Scalars['String'];
};

export type Cluster = {
    __typename?: 'Cluster';
    admissionController: Scalars['Boolean'];
    admissionControllerEvents: Scalars['Boolean'];
    admissionControllerUpdates: Scalars['Boolean'];
    alertCount: Scalars['Int'];
    alerts: Array<Alert>;
    centralApiEndpoint: Scalars['String'];
    clusterVulnerabilities: Array<ClusterVulnerability>;
    clusterVulnerabilityCount: Scalars['Int'];
    clusterVulnerabilityCounter: VulnerabilityCounter;
    collectionMethod: CollectionMethod;
    collectorImage: Scalars['String'];
    complianceControlCount: ComplianceControlCount;
    complianceResults: Array<ControlResult>;
    controlStatus: Scalars['String'];
    controls: Array<ComplianceControl>;
    deploymentCount: Scalars['Int'];
    deployments: Array<Deployment>;
    dynamicConfig?: Maybe<DynamicClusterConfig>;
    failingControls: Array<ComplianceControl>;
    failingPolicyCounter?: Maybe<PolicyCounter>;
    healthStatus?: Maybe<ClusterHealthStatus>;
    helmConfig?: Maybe<CompleteClusterConfig>;
    id: Scalars['ID'];
    imageComponentCount: Scalars['Int'];
    imageComponents: Array<ImageComponent>;
    imageCount: Scalars['Int'];
    imageVulnerabilities: Array<ImageVulnerability>;
    imageVulnerabilityCount: Scalars['Int'];
    imageVulnerabilityCounter: VulnerabilityCounter;
    images: Array<Image>;
    initBundleId: Scalars['String'];
    isGKECluster: Scalars['Boolean'];
    isOpenShiftCluster: Scalars['Boolean'];
    istioClusterVulnerabilities: Array<ClusterVulnerability>;
    istioClusterVulnerabilityCount: Scalars['Int'];
    istioEnabled: Scalars['Boolean'];
    k8sClusterVulnerabilities: Array<ClusterVulnerability>;
    k8sClusterVulnerabilityCount: Scalars['Int'];
    k8sRole?: Maybe<K8SRole>;
    k8sRoleCount: Scalars['Int'];
    k8sRoles: Array<K8SRole>;
    labels: Array<Label>;
    latestViolation?: Maybe<Scalars['Time']>;
    mainImage: Scalars['String'];
    managedBy: ManagerType;
    mostRecentSensorId?: Maybe<SensorDeploymentIdentification>;
    name: Scalars['String'];
    namespace?: Maybe<Namespace>;
    namespaceCount: Scalars['Int'];
    namespaces: Array<Namespace>;
    node?: Maybe<Node>;
    nodeComponentCount: Scalars['Int'];
    nodeComponents: Array<NodeComponent>;
    nodeCount: Scalars['Int'];
    nodeVulnerabilities: Array<NodeVulnerability>;
    nodeVulnerabilityCount: Scalars['Int'];
    nodeVulnerabilityCounter: VulnerabilityCounter;
    nodes: Array<Node>;
    openShiftClusterVulnerabilities: Array<ClusterVulnerability>;
    openShiftClusterVulnerabilityCount: Scalars['Int'];
    passingControls: Array<ComplianceControl>;
    plottedImageVulnerabilities: PlottedImageVulnerabilities;
    plottedNodeVulnerabilities: PlottedNodeVulnerabilities;
    policies: Array<Policy>;
    policyCount: Scalars['Int'];
    policyStatus: PolicyStatus;
    priority: Scalars['Int'];
    risk?: Maybe<Risk>;
    runtimeSupport: Scalars['Boolean'];
    secretCount: Scalars['Int'];
    secrets: Array<Secret>;
    serviceAccount?: Maybe<ServiceAccount>;
    serviceAccountCount: Scalars['Int'];
    serviceAccounts: Array<ServiceAccount>;
    slimCollector: Scalars['Boolean'];
    status?: Maybe<ClusterStatus>;
    subject?: Maybe<Subject>;
    subjectCount: Scalars['Int'];
    subjects: Array<Subject>;
    tolerationsConfig?: Maybe<TolerationsConfig>;
    type: ClusterType;
    unusedVarSink?: Maybe<Scalars['Int']>;
};

export type ClusterAlertCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterAlertsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterClusterVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type ClusterClusterVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterClusterVulnerabilityCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterComplianceControlCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterComplianceResultsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterControlStatusArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterControlsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterFailingControlsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterFailingPolicyCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterImageComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterImageComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterImageCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterImageVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type ClusterImageVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterImageVulnerabilityCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterImagesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterIstioClusterVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterIstioClusterVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterK8sClusterVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterK8sClusterVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterK8sRoleArgs = {
    role: Scalars['ID'];
};

export type ClusterK8sRoleCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterK8sRolesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterLatestViolationArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterNamespaceArgs = {
    name: Scalars['String'];
};

export type ClusterNamespaceCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterNamespacesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterNodeArgs = {
    node: Scalars['ID'];
};

export type ClusterNodeComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterNodeComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterNodeCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterNodeVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type ClusterNodeVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterNodeVulnerabilityCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterNodesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterOpenShiftClusterVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterOpenShiftClusterVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterPassingControlsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterPlottedImageVulnerabilitiesArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterPlottedNodeVulnerabilitiesArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterPoliciesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterPolicyCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterPolicyStatusArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterSecretCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterSecretsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterServiceAccountArgs = {
    sa: Scalars['ID'];
};

export type ClusterServiceAccountCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterServiceAccountsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterSubjectArgs = {
    name: Scalars['String'];
};

export type ClusterSubjectCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterSubjectsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterCve = {
    __typename?: 'ClusterCVE';
    cveBaseInfo?: Maybe<CveInfo>;
    cvss: Scalars['Float'];
    id: Scalars['ID'];
    impactScore: Scalars['Float'];
    severity: VulnerabilitySeverity;
    snoozeExpiry?: Maybe<Scalars['Time']>;
    snoozeStart?: Maybe<Scalars['Time']>;
    snoozed: Scalars['Boolean'];
    type: Cve_CveType;
};

export type ClusterCertExpiryStatus = {
    __typename?: 'ClusterCertExpiryStatus';
    sensorCertExpiry?: Maybe<Scalars['Time']>;
    sensorCertNotBefore?: Maybe<Scalars['Time']>;
};

export type ClusterHealthCounter = {
    __typename?: 'ClusterHealthCounter';
    degraded: Scalars['Int'];
    healthy: Scalars['Int'];
    total: Scalars['Int'];
    unhealthy: Scalars['Int'];
    uninitialized: Scalars['Int'];
};

export type ClusterHealthStatus = {
    __typename?: 'ClusterHealthStatus';
    admissionControlHealthInfo?: Maybe<AdmissionControlHealthInfo>;
    admissionControlHealthStatus: ClusterHealthStatus_HealthStatusLabel;
    collectorHealthInfo?: Maybe<CollectorHealthInfo>;
    collectorHealthStatus: ClusterHealthStatus_HealthStatusLabel;
    healthInfoComplete: Scalars['Boolean'];
    id: Scalars['ID'];
    lastContact?: Maybe<Scalars['Time']>;
    overallHealthStatus: ClusterHealthStatus_HealthStatusLabel;
    scannerHealthInfo?: Maybe<ScannerHealthInfo>;
    scannerHealthStatus: ClusterHealthStatus_HealthStatusLabel;
    sensorHealthStatus: ClusterHealthStatus_HealthStatusLabel;
};

export enum ClusterHealthStatus_HealthStatusLabel {
    Degraded = 'DEGRADED',
    Healthy = 'HEALTHY',
    Unavailable = 'UNAVAILABLE',
    Unhealthy = 'UNHEALTHY',
    Uninitialized = 'UNINITIALIZED',
}

export type ClusterStatus = {
    __typename?: 'ClusterStatus';
    certExpiryStatus?: Maybe<ClusterCertExpiryStatus>;
    orchestratorMetadata?: Maybe<OrchestratorMetadata>;
    providerMetadata?: Maybe<ProviderMetadata>;
    sensorVersion: Scalars['String'];
    upgradeStatus?: Maybe<ClusterUpgradeStatus>;
};

export enum ClusterType {
    GenericCluster = 'GENERIC_CLUSTER',
    KubernetesCluster = 'KUBERNETES_CLUSTER',
    Openshift4Cluster = 'OPENSHIFT4_CLUSTER',
    OpenshiftCluster = 'OPENSHIFT_CLUSTER',
}

export type ClusterUpgradeStatus = {
    __typename?: 'ClusterUpgradeStatus';
    mostRecentProcess?: Maybe<ClusterUpgradeStatus_UpgradeProcessStatus>;
    upgradability: ClusterUpgradeStatus_Upgradability;
    upgradabilityStatusReason: Scalars['String'];
};

export enum ClusterUpgradeStatus_Upgradability {
    AutoUpgradePossible = 'AUTO_UPGRADE_POSSIBLE',
    ManualUpgradeRequired = 'MANUAL_UPGRADE_REQUIRED',
    SensorVersionHigher = 'SENSOR_VERSION_HIGHER',
    Unset = 'UNSET',
    UpToDate = 'UP_TO_DATE',
}

export type ClusterUpgradeStatus_UpgradeProcessStatus = {
    __typename?: 'ClusterUpgradeStatus_UpgradeProcessStatus';
    active: Scalars['Boolean'];
    id: Scalars['ID'];
    initiatedAt?: Maybe<Scalars['Time']>;
    progress?: Maybe<UpgradeProgress>;
    targetVersion: Scalars['String'];
    type: ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType;
    upgraderImage: Scalars['String'];
};

export enum ClusterUpgradeStatus_UpgradeProcessStatus_UpgradeProcessType {
    CertRotation = 'CERT_ROTATION',
    Upgrade = 'UPGRADE',
}

export type ClusterVulnerability = {
    __typename?: 'ClusterVulnerability';
    clusterCount: Scalars['Int'];
    clusters: Array<Cluster>;
    createdAt?: Maybe<Scalars['Time']>;
    cve: Scalars['String'];
    cveBaseInfo?: Maybe<CveInfo>;
    cvss: Scalars['Float'];
    envImpact: Scalars['Float'];
    fixedByVersion: Scalars['String'];
    id: Scalars['ID'];
    impactScore: Scalars['Float'];
    isFixable: Scalars['Boolean'];
    lastModified?: Maybe<Scalars['Time']>;
    lastScanned?: Maybe<Scalars['Time']>;
    link: Scalars['String'];
    publishedOn?: Maybe<Scalars['Time']>;
    scoreVersion: Scalars['String'];
    severity: Scalars['String'];
    summary: Scalars['String'];
    suppressActivation?: Maybe<Scalars['Time']>;
    suppressExpiry?: Maybe<Scalars['Time']>;
    suppressed: Scalars['Boolean'];
    unusedVarSink?: Maybe<Scalars['Int']>;
    vectors?: Maybe<EmbeddedVulnerabilityVectors>;
    vulnerabilityType: Scalars['String'];
    vulnerabilityTypes: Array<Scalars['String']>;
};

export type ClusterVulnerabilityClusterCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterVulnerabilityClustersArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterVulnerabilityIsFixableArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ClusterVulnerabilityUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export enum CollectionMethod {
    CoreBpf = 'CORE_BPF',
    Ebpf = 'EBPF',
    KernelModule = 'KERNEL_MODULE',
    NoCollection = 'NO_COLLECTION',
    UnsetCollection = 'UNSET_COLLECTION',
}

export type CollectorHealthInfo = {
    __typename?: 'CollectorHealthInfo';
    statusErrors: Array<Scalars['String']>;
    version: Scalars['String'];
};

export type CompleteClusterConfig = {
    __typename?: 'CompleteClusterConfig';
    clusterLabels: Array<Label>;
    configFingerprint: Scalars['String'];
    dynamicConfig?: Maybe<DynamicClusterConfig>;
    staticConfig?: Maybe<StaticClusterConfig>;
};

export type ComplianceAggregation_AggregationKey = {
    __typename?: 'ComplianceAggregation_AggregationKey';
    id: Scalars['ID'];
    scope: ComplianceAggregation_Scope;
};

export type ComplianceAggregation_Response = {
    __typename?: 'ComplianceAggregation_Response';
    errorMessage: Scalars['String'];
    results: Array<Maybe<ComplianceAggregation_Result>>;
    sources: Array<Maybe<ComplianceAggregation_Source>>;
};

export type ComplianceAggregation_Result = {
    __typename?: 'ComplianceAggregation_Result';
    aggregationKeys: Array<Maybe<ComplianceAggregation_AggregationKey>>;
    keys: Array<ComplianceDomainKey>;
    numFailing: Scalars['Int'];
    numPassing: Scalars['Int'];
    numSkipped: Scalars['Int'];
    unit: ComplianceAggregation_Scope;
};

export enum ComplianceAggregation_Scope {
    Category = 'CATEGORY',
    Check = 'CHECK',
    Cluster = 'CLUSTER',
    Control = 'CONTROL',
    Deployment = 'DEPLOYMENT',
    Namespace = 'NAMESPACE',
    Node = 'NODE',
    Standard = 'STANDARD',
    Unknown = 'UNKNOWN',
}

export type ComplianceAggregation_Source = {
    __typename?: 'ComplianceAggregation_Source';
    clusterId: Scalars['String'];
    failedRuns: Array<Maybe<ComplianceRunMetadata>>;
    standardId: Scalars['String'];
    successfulRun?: Maybe<ComplianceRunMetadata>;
};

export type ComplianceControl = {
    __typename?: 'ComplianceControl';
    complianceControlEntities: Array<Node>;
    complianceControlFailingNodes: Array<Node>;
    complianceControlNodeCount?: Maybe<ComplianceControlNodeCount>;
    complianceControlNodes: Array<Node>;
    complianceControlPassingNodes: Array<Node>;
    complianceResults: Array<ControlResult>;
    description: Scalars['String'];
    groupId: Scalars['String'];
    id: Scalars['ID'];
    implemented: Scalars['Boolean'];
    interpretationText: Scalars['String'];
    name: Scalars['String'];
    standardId: Scalars['String'];
};

export type ComplianceControlComplianceControlEntitiesArgs = {
    clusterID: Scalars['ID'];
};

export type ComplianceControlComplianceControlFailingNodesArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ComplianceControlComplianceControlNodeCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ComplianceControlComplianceControlNodesArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ComplianceControlComplianceControlPassingNodesArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ComplianceControlComplianceResultsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ComplianceControlCount = {
    __typename?: 'ComplianceControlCount';
    failingCount: Scalars['Int'];
    passingCount: Scalars['Int'];
    unknownCount: Scalars['Int'];
};

export type ComplianceControlGroup = {
    __typename?: 'ComplianceControlGroup';
    description: Scalars['String'];
    id: Scalars['ID'];
    name: Scalars['String'];
    numImplementedChecks: Scalars['Int'];
    standardId: Scalars['String'];
};

export type ComplianceControlNodeCount = {
    __typename?: 'ComplianceControlNodeCount';
    failingCount: Scalars['Int'];
    passingCount: Scalars['Int'];
    unknownCount: Scalars['Int'];
};

export type ComplianceControlResult = {
    __typename?: 'ComplianceControlResult';
    controlId: Scalars['String'];
    resource?: Maybe<ComplianceResource>;
    value?: Maybe<ComplianceResultValue>;
};

export type ComplianceControlWithControlStatus = {
    __typename?: 'ComplianceControlWithControlStatus';
    complianceControl: ComplianceControl;
    controlStatus: Scalars['String'];
};

export type ComplianceDomainKey =
    | ComplianceControl
    | ComplianceControlGroup
    | ComplianceDomain_Cluster
    | ComplianceDomain_Deployment
    | ComplianceDomain_Node
    | ComplianceStandardMetadata
    | Namespace;

export type ComplianceDomain_Cluster = {
    __typename?: 'ComplianceDomain_Cluster';
    id: Scalars['ID'];
    name: Scalars['String'];
};

export type ComplianceDomain_Deployment = {
    __typename?: 'ComplianceDomain_Deployment';
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    id: Scalars['ID'];
    name: Scalars['String'];
    namespace: Scalars['String'];
    namespaceId: Scalars['String'];
    type: Scalars['String'];
};

export type ComplianceDomain_Node = {
    __typename?: 'ComplianceDomain_Node';
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    id: Scalars['ID'];
    name: Scalars['String'];
};

export type ComplianceResource = {
    __typename?: 'ComplianceResource';
    cluster?: Maybe<ComplianceResource_ClusterName>;
    deployment?: Maybe<ComplianceResource_DeploymentName>;
    image?: Maybe<ImageName>;
    node?: Maybe<ComplianceResource_NodeName>;
    resource?: Maybe<ComplianceResourceResource>;
};

export type ComplianceResourceResource =
    | ComplianceResource_ClusterName
    | ComplianceResource_DeploymentName
    | ComplianceResource_NodeName
    | ImageName;

export type ComplianceResource_ClusterName = {
    __typename?: 'ComplianceResource_ClusterName';
    id: Scalars['ID'];
    name: Scalars['String'];
};

export type ComplianceResource_DeploymentName = {
    __typename?: 'ComplianceResource_DeploymentName';
    cluster?: Maybe<ComplianceResource_ClusterName>;
    id: Scalars['ID'];
    name: Scalars['String'];
    namespace: Scalars['String'];
};

export type ComplianceResource_NodeName = {
    __typename?: 'ComplianceResource_NodeName';
    cluster?: Maybe<ComplianceResource_ClusterName>;
    id: Scalars['ID'];
    name: Scalars['String'];
};

export type ComplianceResultValue = {
    __typename?: 'ComplianceResultValue';
    evidence: Array<Maybe<ComplianceResultValue_Evidence>>;
    overallState: ComplianceState;
};

export type ComplianceResultValue_Evidence = {
    __typename?: 'ComplianceResultValue_Evidence';
    message: Scalars['String'];
    messageId: Scalars['Int'];
    state: ComplianceState;
};

export type ComplianceRun = {
    __typename?: 'ComplianceRun';
    clusterId: Scalars['String'];
    errorMessage: Scalars['String'];
    finishTime?: Maybe<Scalars['Time']>;
    id: Scalars['ID'];
    standardId: Scalars['String'];
    startTime?: Maybe<Scalars['Time']>;
    state: ComplianceRun_State;
};

export type ComplianceRunMetadata = {
    __typename?: 'ComplianceRunMetadata';
    clusterId: Scalars['String'];
    domainId: Scalars['String'];
    errorMessage: Scalars['String'];
    finishTimestamp?: Maybe<Scalars['Time']>;
    runId: Scalars['String'];
    standardId: Scalars['String'];
    startTimestamp?: Maybe<Scalars['Time']>;
    success: Scalars['Boolean'];
};

export enum ComplianceRun_State {
    EvalutingChecks = 'EVALUTING_CHECKS',
    Finished = 'FINISHED',
    Invalid = 'INVALID',
    Ready = 'READY',
    Started = 'STARTED',
    WaitForData = 'WAIT_FOR_DATA',
}

export type ComplianceStandard = {
    __typename?: 'ComplianceStandard';
    controls: Array<Maybe<ComplianceControl>>;
    groups: Array<Maybe<ComplianceControlGroup>>;
    metadata?: Maybe<ComplianceStandardMetadata>;
};

export type ComplianceStandardMetadata = {
    __typename?: 'ComplianceStandardMetadata';
    complianceResults: Array<ControlResult>;
    controls: Array<ComplianceControl>;
    description: Scalars['String'];
    dynamic: Scalars['Boolean'];
    groups: Array<ComplianceControlGroup>;
    hideScanResults: Scalars['Boolean'];
    id: Scalars['ID'];
    name: Scalars['String'];
    numImplementedChecks: Scalars['Int'];
    scopes: Array<ComplianceStandardMetadata_Scope>;
};

export type ComplianceStandardMetadataComplianceResultsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export enum ComplianceStandardMetadata_Scope {
    Cluster = 'CLUSTER',
    Deployment = 'DEPLOYMENT',
    Namespace = 'NAMESPACE',
    Node = 'NODE',
    Unset = 'UNSET',
}

export enum ComplianceState {
    ComplianceStateError = 'COMPLIANCE_STATE_ERROR',
    ComplianceStateFailure = 'COMPLIANCE_STATE_FAILURE',
    ComplianceStateNote = 'COMPLIANCE_STATE_NOTE',
    ComplianceStateSkip = 'COMPLIANCE_STATE_SKIP',
    ComplianceStateSuccess = 'COMPLIANCE_STATE_SUCCESS',
    ComplianceStateUnknown = 'COMPLIANCE_STATE_UNKNOWN',
}

export type Container = {
    __typename?: 'Container';
    config?: Maybe<ContainerConfig>;
    id: Scalars['ID'];
    image?: Maybe<ContainerImage>;
    livenessProbe?: Maybe<LivenessProbe>;
    name: Scalars['String'];
    ports: Array<Maybe<PortConfig>>;
    readinessProbe?: Maybe<ReadinessProbe>;
    resources?: Maybe<Resources>;
    secrets: Array<Maybe<EmbeddedSecret>>;
    securityContext?: Maybe<SecurityContext>;
    volumes: Array<Maybe<Volume>>;
};

export type ContainerConfig = {
    __typename?: 'ContainerConfig';
    appArmorProfile: Scalars['String'];
    args: Array<Scalars['String']>;
    command: Array<Scalars['String']>;
    directory: Scalars['String'];
    env: Array<Maybe<ContainerConfig_EnvironmentConfig>>;
    uid: Scalars['Int'];
    user: Scalars['String'];
};

export type ContainerConfig_EnvironmentConfig = {
    __typename?: 'ContainerConfig_EnvironmentConfig';
    envVarSource: ContainerConfig_EnvironmentConfig_EnvVarSource;
    key: Scalars['String'];
    value: Scalars['String'];
};

export enum ContainerConfig_EnvironmentConfig_EnvVarSource {
    ConfigMapKey = 'CONFIG_MAP_KEY',
    Field = 'FIELD',
    Raw = 'RAW',
    ResourceField = 'RESOURCE_FIELD',
    SecretKey = 'SECRET_KEY',
    Unknown = 'UNKNOWN',
    Unset = 'UNSET',
}

export type ContainerImage = {
    __typename?: 'ContainerImage';
    id: Scalars['ID'];
    isClusterLocal: Scalars['Boolean'];
    name?: Maybe<ImageName>;
    notPullable: Scalars['Boolean'];
};

export type ContainerInstance = {
    __typename?: 'ContainerInstance';
    containerIps: Array<Scalars['String']>;
    containerName: Scalars['String'];
    containingPodId: Scalars['String'];
    exitCode: Scalars['Int'];
    finished?: Maybe<Scalars['Time']>;
    imageDigest: Scalars['String'];
    instanceId?: Maybe<ContainerInstanceId>;
    started?: Maybe<Scalars['Time']>;
    terminationReason: Scalars['String'];
};

export type ContainerInstanceId = {
    __typename?: 'ContainerInstanceID';
    containerRuntime: ContainerRuntime;
    id: Scalars['ID'];
    node: Scalars['String'];
};

export type ContainerNameGroup = {
    __typename?: 'ContainerNameGroup';
    containerInstances: Array<ContainerInstance>;
    events: Array<DeploymentEvent>;
    id: Scalars['ID'];
    name: Scalars['String'];
    podId: Scalars['String'];
    startTime?: Maybe<Scalars['Time']>;
};

export type ContainerRestartEvent = DeploymentEvent & {
    __typename?: 'ContainerRestartEvent';
    id: Scalars['ID'];
    name: Scalars['String'];
    timestamp?: Maybe<Scalars['Time']>;
};

export enum ContainerRuntime {
    CrioContainerRuntime = 'CRIO_CONTAINER_RUNTIME',
    DockerContainerRuntime = 'DOCKER_CONTAINER_RUNTIME',
    UnknownContainerRuntime = 'UNKNOWN_CONTAINER_RUNTIME',
}

export type ContainerRuntimeInfo = {
    __typename?: 'ContainerRuntimeInfo';
    type: ContainerRuntime;
    version: Scalars['String'];
};

export type ContainerTerminationEvent = DeploymentEvent & {
    __typename?: 'ContainerTerminationEvent';
    exitCode: Scalars['Int'];
    id: Scalars['ID'];
    name: Scalars['String'];
    reason: Scalars['String'];
    timestamp?: Maybe<Scalars['Time']>;
};

export type ControlResult = {
    __typename?: 'ControlResult';
    control?: Maybe<ComplianceControl>;
    resource?: Maybe<Resource>;
    value?: Maybe<ComplianceResultValue>;
};

export type CosignSignature = {
    __typename?: 'CosignSignature';
};

export type DataSource = {
    __typename?: 'DataSource';
    id: Scalars['ID'];
    name: Scalars['String'];
};

export type DeferVulnRequest = {
    comment?: InputMaybe<Scalars['String']>;
    cve?: InputMaybe<Scalars['String']>;
    expiresOn?: InputMaybe<Scalars['Time']>;
    expiresWhenFixed?: InputMaybe<Scalars['Boolean']>;
    scope?: InputMaybe<VulnReqScope>;
};

export type DeferralRequest = {
    __typename?: 'DeferralRequest';
    expiresOn?: Maybe<Scalars['Time']>;
    expiresWhenFixed: Scalars['Boolean'];
};

export type Deployment = {
    __typename?: 'Deployment';
    annotations: Array<Label>;
    automountServiceAccountToken: Scalars['Boolean'];
    cluster?: Maybe<Cluster>;
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    complianceResults: Array<ControlResult>;
    containerRestartCount: Scalars['Int'];
    containerTerminationCount: Scalars['Int'];
    containers: Array<Maybe<Container>>;
    created?: Maybe<Scalars['Time']>;
    deployAlertCount: Scalars['Int'];
    deployAlerts: Array<Alert>;
    failingPolicies: Array<Policy>;
    failingPolicyCount: Scalars['Int'];
    failingPolicyCounter?: Maybe<PolicyCounter>;
    failingRuntimePolicyCount: Scalars['Int'];
    groupedProcesses: Array<ProcessNameGroup>;
    hostIpc: Scalars['Boolean'];
    hostNetwork: Scalars['Boolean'];
    hostPid: Scalars['Boolean'];
    id: Scalars['ID'];
    imageCVECountBySeverity: ResourceCountByCveSeverity;
    imageComponentCount: Scalars['Int'];
    imageComponents: Array<ImageComponent>;
    imageCount: Scalars['Int'];
    imagePullSecrets: Array<Scalars['String']>;
    imageVulnerabilities: Array<ImageVulnerability>;
    imageVulnerabilityCount: Scalars['Int'];
    imageVulnerabilityCounter: VulnerabilityCounter;
    images: Array<Image>;
    inactive: Scalars['Boolean'];
    labelSelector?: Maybe<LabelSelector>;
    labels: Array<Label>;
    latestViolation?: Maybe<Scalars['Time']>;
    name: Scalars['String'];
    namespace: Scalars['String'];
    namespaceId: Scalars['String'];
    namespaceObject?: Maybe<Namespace>;
    orchestratorComponent: Scalars['Boolean'];
    plottedImageVulnerabilities: PlottedImageVulnerabilities;
    podCount: Scalars['Int'];
    podLabels: Array<Label>;
    policies: Array<Policy>;
    policyCount: Scalars['Int'];
    policyStatus: Scalars['String'];
    ports: Array<Maybe<PortConfig>>;
    priority: Scalars['Int'];
    processActivityCount: Scalars['Int'];
    replicas: Scalars['Int'];
    riskScore: Scalars['Float'];
    runtimeClass: Scalars['String'];
    secretCount: Scalars['Int'];
    secrets: Array<Secret>;
    serviceAccount: Scalars['String'];
    serviceAccountID: Scalars['String'];
    serviceAccountObject?: Maybe<ServiceAccount>;
    serviceAccountPermissionLevel: PermissionLevel;
    stateTimestamp: Scalars['Int'];
    tolerations: Array<Maybe<Toleration>>;
    type: Scalars['String'];
    unusedVarSink?: Maybe<Scalars['Int']>;
};

export type DeploymentComplianceResultsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentDeployAlertCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentDeployAlertsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentFailingPoliciesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentFailingPolicyCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentFailingPolicyCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentFailingRuntimePolicyCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentImageCveCountBySeverityArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentImageComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentImageComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentImageCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentImageVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type DeploymentImageVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentImageVulnerabilityCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentImagesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentLatestViolationArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentPlottedImageVulnerabilitiesArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentPoliciesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentPolicyCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentPolicyStatusArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentSecretCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentSecretsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type DeploymentEvent = {
    id: Scalars['ID'];
    name: Scalars['String'];
    timestamp?: Maybe<Scalars['Time']>;
};

export type DeploymentsWithMostSevereViolations = {
    __typename?: 'DeploymentsWithMostSevereViolations';
    clusterName: Scalars['String'];
    failingPolicySeverityCounts: PolicyCounter;
    id: Scalars['ID'];
    name: Scalars['String'];
    namespace: Scalars['String'];
};

export type DynamicClusterConfig = {
    __typename?: 'DynamicClusterConfig';
    admissionControllerConfig?: Maybe<AdmissionControllerConfig>;
    disableAuditLogs: Scalars['Boolean'];
    registryOverride: Scalars['String'];
};

export type Email = {
    __typename?: 'Email';
    allowUnauthenticatedSmtp: Scalars['Boolean'];
    disableTLS: Scalars['Boolean'];
    from: Scalars['String'];
    password: Scalars['String'];
    sender: Scalars['String'];
    server: Scalars['String'];
    startTLSAuthMethod: Email_AuthMethod;
    username: Scalars['String'];
};

export enum Email_AuthMethod {
    Disabled = 'DISABLED',
    Login = 'LOGIN',
    Plain = 'PLAIN',
}

export type EmbeddedImageScanComponent = {
    __typename?: 'EmbeddedImageScanComponent';
    activeState?: Maybe<ActiveState>;
    deploymentCount: Scalars['Int'];
    deployments: Array<Deployment>;
    fixedIn: Scalars['String'];
    id: Scalars['ID'];
    imageCount: Scalars['Int'];
    images: Array<Image>;
    lastScanned?: Maybe<Scalars['Time']>;
    layerIndex?: Maybe<Scalars['Int']>;
    license?: Maybe<License>;
    location: Scalars['String'];
    name: Scalars['String'];
    nodeCount: Scalars['Int'];
    nodes: Array<Node>;
    priority: Scalars['Int'];
    riskScore: Scalars['Float'];
    source: Scalars['String'];
    topVuln?: Maybe<EmbeddedVulnerability>;
    unusedVarSink?: Maybe<Scalars['Int']>;
    version: Scalars['String'];
    vulnCount: Scalars['Int'];
    vulnCounter: VulnerabilityCounter;
    vulns: Array<Maybe<EmbeddedVulnerability>>;
};

export type EmbeddedImageScanComponentActiveStateArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedImageScanComponentDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type EmbeddedImageScanComponentDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type EmbeddedImageScanComponentImageCountArgs = {
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type EmbeddedImageScanComponentImagesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type EmbeddedImageScanComponentLocationArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedImageScanComponentNodeCountArgs = {
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type EmbeddedImageScanComponentNodesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type EmbeddedImageScanComponentUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedImageScanComponentVulnCountArgs = {
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type EmbeddedImageScanComponentVulnCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedImageScanComponentVulnsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type EmbeddedImageScanComponent_Executable = {
    __typename?: 'EmbeddedImageScanComponent_Executable';
    dependencies: Array<Scalars['String']>;
    path: Scalars['String'];
};

export type EmbeddedNodeScanComponent = {
    __typename?: 'EmbeddedNodeScanComponent';
    id: Scalars['ID'];
    lastScanned?: Maybe<Scalars['Time']>;
    name: Scalars['String'];
    priority: Scalars['Int'];
    riskScore: Scalars['Float'];
    topVuln?: Maybe<EmbeddedVulnerability>;
    unusedVarSink?: Maybe<Scalars['Int']>;
    version: Scalars['String'];
    vulnCount: Scalars['Int'];
    vulnCounter: VulnerabilityCounter;
    vulns: Array<Maybe<EmbeddedVulnerability>>;
};

export type EmbeddedNodeScanComponentUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedNodeScanComponentVulnCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedNodeScanComponentVulnCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedNodeScanComponentVulnsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedSecret = {
    __typename?: 'EmbeddedSecret';
    name: Scalars['String'];
    path: Scalars['String'];
};

export type EmbeddedVulnerability = {
    __typename?: 'EmbeddedVulnerability';
    activeState?: Maybe<ActiveState>;
    createdAt?: Maybe<Scalars['Time']>;
    cve: Scalars['String'];
    cvss: Scalars['Float'];
    deploymentCount: Scalars['Int'];
    deployments: Array<Deployment>;
    discoveredAtImage?: Maybe<Scalars['Time']>;
    effectiveVulnerabilityRequest?: Maybe<VulnerabilityRequest>;
    envImpact: Scalars['Float'];
    fixedByVersion: Scalars['String'];
    id: Scalars['ID'];
    imageCount: Scalars['Int'];
    images: Array<Image>;
    impactScore: Scalars['Float'];
    isFixable: Scalars['Boolean'];
    lastModified?: Maybe<Scalars['Time']>;
    lastScanned?: Maybe<Scalars['Time']>;
    link: Scalars['String'];
    nodeCount: Scalars['Int'];
    nodes: Array<Node>;
    publishedOn?: Maybe<Scalars['Time']>;
    scoreVersion: Scalars['String'];
    severity: Scalars['String'];
    summary: Scalars['String'];
    suppressActivation?: Maybe<Scalars['Time']>;
    suppressExpiry?: Maybe<Scalars['Time']>;
    suppressed: Scalars['Boolean'];
    vectors?: Maybe<EmbeddedVulnerabilityVectors>;
    vulnerabilityState: Scalars['String'];
    vulnerabilityType: Scalars['String'];
    vulnerabilityTypes: Array<Scalars['String']>;
};

export type EmbeddedVulnerabilityActiveStateArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedVulnerabilityDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedVulnerabilityDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedVulnerabilityDiscoveredAtImageArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedVulnerabilityImageCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedVulnerabilityImagesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedVulnerabilityIsFixableArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedVulnerabilityNodeCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedVulnerabilityNodesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type EmbeddedVulnerabilityVectors = Cvssv2 | Cvssv3;

export enum EmbeddedVulnerability_ScoreVersion {
    V2 = 'V2',
    V3 = 'V3',
}

export enum EmbeddedVulnerability_VulnerabilityType {
    ImageVulnerability = 'IMAGE_VULNERABILITY',
    IstioVulnerability = 'ISTIO_VULNERABILITY',
    K8SVulnerability = 'K8S_VULNERABILITY',
    NodeVulnerability = 'NODE_VULNERABILITY',
    OpenshiftVulnerability = 'OPENSHIFT_VULNERABILITY',
    UnknownVulnerability = 'UNKNOWN_VULNERABILITY',
}

export enum EnforcementAction {
    FailBuildEnforcement = 'FAIL_BUILD_ENFORCEMENT',
    FailDeploymentCreateEnforcement = 'FAIL_DEPLOYMENT_CREATE_ENFORCEMENT',
    FailDeploymentUpdateEnforcement = 'FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT',
    FailKubeRequestEnforcement = 'FAIL_KUBE_REQUEST_ENFORCEMENT',
    KillPodEnforcement = 'KILL_POD_ENFORCEMENT',
    ScaleToZeroEnforcement = 'SCALE_TO_ZERO_ENFORCEMENT',
    UnsatisfiableNodeConstraintEnforcement = 'UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT',
    UnsetEnforcement = 'UNSET_ENFORCEMENT',
}

export enum EventSource {
    AuditLogEvent = 'AUDIT_LOG_EVENT',
    DeploymentEvent = 'DEPLOYMENT_EVENT',
    NotApplicable = 'NOT_APPLICABLE',
}

export type Exclusion = {
    __typename?: 'Exclusion';
    deployment?: Maybe<Exclusion_Deployment>;
    expiration?: Maybe<Scalars['Time']>;
    image?: Maybe<Exclusion_Image>;
    name: Scalars['String'];
};

export type Exclusion_Deployment = {
    __typename?: 'Exclusion_Deployment';
    name: Scalars['String'];
    scope?: Maybe<Scope>;
};

export type Exclusion_Image = {
    __typename?: 'Exclusion_Image';
    name: Scalars['String'];
};

export type FalsePositiveRequest = {
    __typename?: 'FalsePositiveRequest';
};

export type FalsePositiveVulnRequest = {
    comment?: InputMaybe<Scalars['String']>;
    cve?: InputMaybe<Scalars['String']>;
    scope?: InputMaybe<VulnReqScope>;
};

export type GenerateTokenResponse = {
    __typename?: 'GenerateTokenResponse';
    metadata?: Maybe<TokenMetadata>;
    token: Scalars['String'];
};

export type Generic = {
    __typename?: 'Generic';
    auditLoggingEnabled: Scalars['Boolean'];
    caCert: Scalars['String'];
    endpoint: Scalars['String'];
    extraFields: Array<Maybe<KeyValuePair>>;
    headers: Array<Maybe<KeyValuePair>>;
    password: Scalars['String'];
    skipTLSVerify: Scalars['Boolean'];
    username: Scalars['String'];
};

export type GetComplianceRunStatusesResponse = {
    __typename?: 'GetComplianceRunStatusesResponse';
    invalidRunIds: Array<Scalars['String']>;
    runs: Array<Maybe<ComplianceRun>>;
};

export type GetPermissionsResponse = {
    __typename?: 'GetPermissionsResponse';
    resourceToAccess: Array<Label>;
};

export type GoogleProviderMetadata = {
    __typename?: 'GoogleProviderMetadata';
    clusterName: Scalars['String'];
    project: Scalars['String'];
};

export type Group = {
    __typename?: 'Group';
    props?: Maybe<GroupProperties>;
    roleName: Scalars['String'];
};

export type GroupProperties = {
    __typename?: 'GroupProperties';
    authProviderId: Scalars['String'];
    id: Scalars['ID'];
    key: Scalars['String'];
    traits?: Maybe<Traits>;
    value: Scalars['String'];
};

export type Image = {
    __typename?: 'Image';
    dataSource?: Maybe<DataSource>;
    deploymentCount: Scalars['Int'];
    deployments: Array<Deployment>;
    id: Scalars['ID'];
    imageCVECountBySeverity: ResourceCountByCveSeverity;
    imageComponentCount: Scalars['Int'];
    imageComponents: Array<ImageComponent>;
    imageVulnerabilities: Array<Maybe<ImageVulnerability>>;
    imageVulnerabilityCount: Scalars['Int'];
    imageVulnerabilityCounter: VulnerabilityCounter;
    isClusterLocal: Scalars['Boolean'];
    lastUpdated?: Maybe<Scalars['Time']>;
    metadata?: Maybe<ImageMetadata>;
    name?: Maybe<ImageName>;
    names: Array<Maybe<ImageName>>;
    notPullable: Scalars['Boolean'];
    notes: Array<Image_Note>;
    operatingSystem: Scalars['String'];
    plottedImageVulnerabilities: PlottedImageVulnerabilities;
    priority: Scalars['Int'];
    riskScore: Scalars['Float'];
    scan?: Maybe<ImageScan>;
    scanNotes: Array<ImageScan_Note>;
    scanTime?: Maybe<Scalars['Time']>;
    scannerVersion: Scalars['String'];
    signature?: Maybe<ImageSignature>;
    signatureVerificationData?: Maybe<ImageSignatureVerificationData>;
    topImageVulnerability?: Maybe<ImageVulnerability>;
    unusedVarSink?: Maybe<Scalars['Int']>;
    watchStatus: ImageWatchStatus;
};

export type ImageDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ImageImageCveCountBySeverityArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageImageComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageImageComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ImageImageVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type ImageImageVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageImageVulnerabilityCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImagePlottedImageVulnerabilitiesArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageTopImageVulnerabilityArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageCve = {
    __typename?: 'ImageCVE';
    cveBaseInfo?: Maybe<CveInfo>;
    cvss: Scalars['Float'];
    id: Scalars['ID'];
    impactScore: Scalars['Float'];
    operatingSystem: Scalars['String'];
    severity: VulnerabilitySeverity;
    snoozeExpiry?: Maybe<Scalars['Time']>;
    snoozeStart?: Maybe<Scalars['Time']>;
    snoozed: Scalars['Boolean'];
};

export type ImageCveCore = {
    __typename?: 'ImageCVECore';
    affectedImageCount: Scalars['Int'];
    affectedImageCountBySeverity: ResourceCountByCveSeverity;
    cve: Scalars['String'];
    deployments: Array<Deployment>;
    distroTuples: Array<ImageVulnerability>;
    firstDiscoveredInSystem?: Maybe<Scalars['Time']>;
    images: Array<Image>;
    topCVSS: Scalars['Float'];
};

export type ImageCveCoreDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
};

export type ImageCveCoreImagesArgs = {
    pagination?: InputMaybe<Pagination>;
};

export type ImageComponent = {
    __typename?: 'ImageComponent';
    activeState?: Maybe<ActiveState>;
    deploymentCount: Scalars['Int'];
    deployments: Array<Deployment>;
    fixedBy: Scalars['String'];
    /** @deprecated use 'fixedBy' */
    fixedIn: Scalars['String'];
    id: Scalars['ID'];
    imageCount: Scalars['Int'];
    imageVulnerabilities: Array<Maybe<ImageVulnerability>>;
    imageVulnerabilityCount: Scalars['Int'];
    imageVulnerabilityCounter: VulnerabilityCounter;
    images: Array<Image>;
    lastScanned?: Maybe<Scalars['Time']>;
    layerIndex?: Maybe<Scalars['Int']>;
    license?: Maybe<License>;
    location: Scalars['String'];
    name: Scalars['String'];
    operatingSystem: Scalars['String'];
    plottedImageVulnerabilities: PlottedImageVulnerabilities;
    priority: Scalars['Int'];
    riskScore: Scalars['Float'];
    source: SourceType;
    topImageVulnerability?: Maybe<ImageVulnerability>;
    unusedVarSink?: Maybe<Scalars['Int']>;
    version: Scalars['String'];
};

export type ImageComponentActiveStateArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageComponentDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type ImageComponentDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type ImageComponentImageCountArgs = {
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type ImageComponentImageVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type ImageComponentImageVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type ImageComponentImageVulnerabilityCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageComponentImagesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type ImageComponentLocationArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageComponentPlottedImageVulnerabilitiesArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageComponentUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageLayer = {
    __typename?: 'ImageLayer';
    author: Scalars['String'];
    created?: Maybe<Scalars['Time']>;
    empty: Scalars['Boolean'];
    instruction: Scalars['String'];
    value: Scalars['String'];
};

export type ImageMetadata = {
    __typename?: 'ImageMetadata';
    dataSource?: Maybe<DataSource>;
    layerShas: Array<Scalars['String']>;
    v1?: Maybe<V1Metadata>;
    v2?: Maybe<V2Metadata>;
};

export type ImageName = {
    __typename?: 'ImageName';
    fullName: Scalars['String'];
    registry: Scalars['String'];
    remote: Scalars['String'];
    tag: Scalars['String'];
};

export type ImagePullSecret = {
    __typename?: 'ImagePullSecret';
    registries: Array<Maybe<ImagePullSecret_Registry>>;
};

export type ImagePullSecret_Registry = {
    __typename?: 'ImagePullSecret_Registry';
    name: Scalars['String'];
    username: Scalars['String'];
};

export type ImageScan = {
    __typename?: 'ImageScan';
    /** @deprecated use 'imageComponentCount' */
    componentCount: Scalars['Int'];
    /** @deprecated use 'imageComponents' */
    components: Array<EmbeddedImageScanComponent>;
    dataSource?: Maybe<DataSource>;
    imageComponentCount: Scalars['Int'];
    imageComponents: Array<ImageComponent>;
    notes: Array<ImageScan_Note>;
    operatingSystem: Scalars['String'];
    scanTime?: Maybe<Scalars['Time']>;
    scannerVersion: Scalars['String'];
};

export type ImageScanComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageScanComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ImageScanImageComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageScanImageComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export enum ImageScan_Note {
    CertifiedRhelScanUnavailable = 'CERTIFIED_RHEL_SCAN_UNAVAILABLE',
    LanguageCvesUnavailable = 'LANGUAGE_CVES_UNAVAILABLE',
    OsCvesStale = 'OS_CVES_STALE',
    OsCvesUnavailable = 'OS_CVES_UNAVAILABLE',
    OsUnavailable = 'OS_UNAVAILABLE',
    PartialScanData = 'PARTIAL_SCAN_DATA',
    Unset = 'UNSET',
}

export type ImageSignature = {
    __typename?: 'ImageSignature';
    fetched?: Maybe<Scalars['Time']>;
    signatures: Array<Maybe<Signature>>;
};

export type ImageSignatureVerificationData = {
    __typename?: 'ImageSignatureVerificationData';
    results: Array<Maybe<ImageSignatureVerificationResult>>;
};

export type ImageSignatureVerificationResult = {
    __typename?: 'ImageSignatureVerificationResult';
    description: Scalars['String'];
    status: ImageSignatureVerificationResult_Status;
    verificationTime?: Maybe<Scalars['Time']>;
    verifiedImageReferences: Array<Scalars['String']>;
    verifierId: Scalars['String'];
};

export enum ImageSignatureVerificationResult_Status {
    CorruptedSignature = 'CORRUPTED_SIGNATURE',
    FailedVerification = 'FAILED_VERIFICATION',
    GenericError = 'GENERIC_ERROR',
    InvalidSignatureAlgo = 'INVALID_SIGNATURE_ALGO',
    Unset = 'UNSET',
    Verified = 'VERIFIED',
}

export type ImageVulnerability = {
    __typename?: 'ImageVulnerability';
    activeState?: Maybe<ActiveState>;
    createdAt?: Maybe<Scalars['Time']>;
    cve: Scalars['String'];
    cveBaseInfo?: Maybe<CveInfo>;
    cvss: Scalars['Float'];
    deploymentCount: Scalars['Int'];
    deployments: Array<Deployment>;
    discoveredAtImage?: Maybe<Scalars['Time']>;
    effectiveVulnerabilityRequest?: Maybe<VulnerabilityRequest>;
    envImpact: Scalars['Float'];
    fixedByVersion: Scalars['String'];
    id: Scalars['ID'];
    imageComponentCount: Scalars['Int'];
    imageComponents: Array<ImageComponent>;
    imageCount: Scalars['Int'];
    images: Array<Image>;
    impactScore: Scalars['Float'];
    isFixable: Scalars['Boolean'];
    lastModified?: Maybe<Scalars['Time']>;
    lastScanned?: Maybe<Scalars['Time']>;
    link: Scalars['String'];
    operatingSystem: Scalars['String'];
    publishedOn?: Maybe<Scalars['Time']>;
    scoreVersion: Scalars['String'];
    severity: Scalars['String'];
    summary: Scalars['String'];
    suppressActivation?: Maybe<Scalars['Time']>;
    suppressExpiry?: Maybe<Scalars['Time']>;
    suppressed: Scalars['Boolean'];
    unusedVarSink?: Maybe<Scalars['Int']>;
    vectors?: Maybe<EmbeddedVulnerabilityVectors>;
    vulnerabilityState: Scalars['String'];
};

export type ImageVulnerabilityActiveStateArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageVulnerabilityDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageVulnerabilityDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ImageVulnerabilityDiscoveredAtImageArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageVulnerabilityImageComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageVulnerabilityImageComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ImageVulnerabilityImageCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageVulnerabilityImagesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ImageVulnerabilityIsFixableArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ImageVulnerabilityUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export enum ImageWatchStatus {
    NotWatched = 'NOT_WATCHED',
    Watched = 'WATCHED',
}

export enum Image_Note {
    MissingMetadata = 'MISSING_METADATA',
    MissingScanData = 'MISSING_SCAN_DATA',
    MissingSignature = 'MISSING_SIGNATURE',
    MissingSignatureVerificationData = 'MISSING_SIGNATURE_VERIFICATION_DATA',
}

export type Jira = {
    __typename?: 'Jira';
    defaultFieldsJson: Scalars['String'];
    issueType: Scalars['String'];
    password: Scalars['String'];
    priorityMappings: Array<Maybe<Jira_PriorityMapping>>;
    url: Scalars['String'];
    username: Scalars['String'];
};

export type Jira_PriorityMapping = {
    __typename?: 'Jira_PriorityMapping';
    priorityName: Scalars['String'];
    severity: Severity;
};

export type K8SRole = {
    __typename?: 'K8SRole';
    annotations: Array<Label>;
    cluster: Cluster;
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    clusterRole: Scalars['Boolean'];
    createdAt?: Maybe<Scalars['Time']>;
    id: Scalars['ID'];
    labels: Array<Label>;
    name: Scalars['String'];
    namespace: Scalars['String'];
    resources: Array<Scalars['String']>;
    roleNamespace?: Maybe<Namespace>;
    rules: Array<Maybe<PolicyRule>>;
    serviceAccountCount: Scalars['Int'];
    serviceAccounts: Array<ServiceAccount>;
    subjectCount: Scalars['Int'];
    subjects: Array<Subject>;
    type: Scalars['String'];
    urls: Array<Scalars['String']>;
    verbs: Array<Scalars['String']>;
};

export type K8SRoleServiceAccountCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type K8SRoleServiceAccountsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type K8SRoleSubjectCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type K8SRoleSubjectsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type K8SRoleBinding = {
    __typename?: 'K8SRoleBinding';
    annotations: Array<Label>;
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    clusterRole: Scalars['Boolean'];
    createdAt?: Maybe<Scalars['Time']>;
    id: Scalars['ID'];
    labels: Array<Label>;
    name: Scalars['String'];
    namespace: Scalars['String'];
    roleId: Scalars['String'];
    subjects: Array<Maybe<Subject>>;
};

export type KeyValuePair = {
    __typename?: 'KeyValuePair';
    key: Scalars['String'];
    value: Scalars['String'];
};

export enum L4Protocol {
    L4ProtocolAny = 'L4_PROTOCOL_ANY',
    L4ProtocolIcmp = 'L4_PROTOCOL_ICMP',
    L4ProtocolRaw = 'L4_PROTOCOL_RAW',
    L4ProtocolSctp = 'L4_PROTOCOL_SCTP',
    L4ProtocolTcp = 'L4_PROTOCOL_TCP',
    L4ProtocolUdp = 'L4_PROTOCOL_UDP',
    L4ProtocolUnknown = 'L4_PROTOCOL_UNKNOWN',
}

export type Label = {
    __typename?: 'Label';
    key: Scalars['String'];
    value: Scalars['String'];
};

export type LabelSelector = {
    __typename?: 'LabelSelector';
    matchLabels: Array<Label>;
    requirements: Array<Maybe<LabelSelector_Requirement>>;
};

export enum LabelSelector_Operator {
    Exists = 'EXISTS',
    In = 'IN',
    NotExists = 'NOT_EXISTS',
    NotIn = 'NOT_IN',
    Unknown = 'UNKNOWN',
}

export type LabelSelector_Requirement = {
    __typename?: 'LabelSelector_Requirement';
    key: Scalars['String'];
    op: LabelSelector_Operator;
    values: Array<Scalars['String']>;
};

export type License = {
    __typename?: 'License';
    name: Scalars['String'];
    type: Scalars['String'];
    url: Scalars['String'];
};

export enum LifecycleStage {
    Build = 'BUILD',
    Deploy = 'DEPLOY',
    Runtime = 'RUNTIME',
}

export enum ListAlert_ResourceType {
    Configmaps = 'CONFIGMAPS',
    Deployment = 'DEPLOYMENT',
    Secrets = 'SECRETS',
}

export type LivenessProbe = {
    __typename?: 'LivenessProbe';
    defined: Scalars['Boolean'];
};

export enum ManagerType {
    ManagerTypeHelmChart = 'MANAGER_TYPE_HELM_CHART',
    ManagerTypeKubernetesOperator = 'MANAGER_TYPE_KUBERNETES_OPERATOR',
    ManagerTypeManual = 'MANAGER_TYPE_MANUAL',
    ManagerTypeUnknown = 'MANAGER_TYPE_UNKNOWN',
}

export type Metadata = {
    __typename?: 'Metadata';
    buildFlavor: Scalars['String'];
    licenseStatus: Metadata_LicenseStatus;
    releaseBuild: Scalars['Boolean'];
    version: Scalars['String'];
};

export enum Metadata_LicenseStatus {
    Expired = 'EXPIRED',
    Invalid = 'INVALID',
    None = 'NONE',
    Restarting = 'RESTARTING',
    Valid = 'VALID',
}

export type MitreAttackVector = {
    __typename?: 'MitreAttackVector';
    tactic?: Maybe<MitreTactic>;
    techniques: Array<Maybe<MitreTechnique>>;
};

export type MitreTactic = {
    __typename?: 'MitreTactic';
    description: Scalars['String'];
    id: Scalars['ID'];
    name: Scalars['String'];
};

export type MitreTechnique = {
    __typename?: 'MitreTechnique';
    description: Scalars['String'];
    id: Scalars['ID'];
    name: Scalars['String'];
};

export type Mutation = {
    __typename?: 'Mutation';
    approveVulnerabilityRequest: VulnerabilityRequest;
    complianceTriggerRuns: Array<ComplianceRun>;
    deferVulnerability: VulnerabilityRequest;
    deleteVulnerabilityRequest: Scalars['Boolean'];
    denyVulnerabilityRequest: VulnerabilityRequest;
    markVulnerabilityFalsePositive: VulnerabilityRequest;
    undoVulnerabilityRequest: VulnerabilityRequest;
    updateVulnerabilityRequest: VulnerabilityRequest;
};

export type MutationApproveVulnerabilityRequestArgs = {
    comment: Scalars['String'];
    requestID: Scalars['ID'];
};

export type MutationComplianceTriggerRunsArgs = {
    clusterId: Scalars['ID'];
    standardId: Scalars['ID'];
};

export type MutationDeferVulnerabilityArgs = {
    request: DeferVulnRequest;
};

export type MutationDeleteVulnerabilityRequestArgs = {
    requestID: Scalars['ID'];
};

export type MutationDenyVulnerabilityRequestArgs = {
    comment: Scalars['String'];
    requestID: Scalars['ID'];
};

export type MutationMarkVulnerabilityFalsePositiveArgs = {
    request: FalsePositiveVulnRequest;
};

export type MutationUndoVulnerabilityRequestArgs = {
    requestID: Scalars['ID'];
};

export type MutationUpdateVulnerabilityRequestArgs = {
    comment: Scalars['String'];
    expiry: VulnReqExpiry;
    requestID: Scalars['ID'];
};

export type Namespace = {
    __typename?: 'Namespace';
    cluster: Cluster;
    complianceResults: Array<ControlResult>;
    deploymentCount: Scalars['Int'];
    deployments: Array<Deployment>;
    failingPolicyCounter?: Maybe<PolicyCounter>;
    imageComponentCount: Scalars['Int'];
    imageComponents: Array<ImageComponent>;
    imageCount: Scalars['Int'];
    imageVulnerabilities: Array<ImageVulnerability>;
    imageVulnerabilityCount: Scalars['Int'];
    imageVulnerabilityCounter: VulnerabilityCounter;
    images: Array<Image>;
    k8sRoleCount: Scalars['Int'];
    k8sRoles: Array<K8SRole>;
    latestViolation?: Maybe<Scalars['Time']>;
    metadata?: Maybe<NamespaceMetadata>;
    networkPolicyCount: Scalars['Int'];
    numDeployments: Scalars['Int'];
    numNetworkPolicies: Scalars['Int'];
    numSecrets: Scalars['Int'];
    plottedImageVulnerabilities: PlottedImageVulnerabilities;
    policies: Array<Policy>;
    policyCount: Scalars['Int'];
    policyStatus: PolicyStatus;
    policyStatusOnly: Scalars['String'];
    risk?: Maybe<Risk>;
    secretCount: Scalars['Int'];
    secrets: Array<Secret>;
    serviceAccountCount: Scalars['Int'];
    serviceAccounts: Array<ServiceAccount>;
    subjectCount: Scalars['Int'];
    subjects: Array<Subject>;
    unusedVarSink?: Maybe<Scalars['Int']>;
};

export type NamespaceComplianceResultsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceFailingPolicyCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceImageComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceImageComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceImageCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceImageVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type NamespaceImageVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceImageVulnerabilityCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceImagesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceK8sRoleCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceK8sRolesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceLatestViolationArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceNetworkPolicyCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespacePlottedImageVulnerabilitiesArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespacePoliciesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type NamespacePolicyCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespacePolicyStatusArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespacePolicyStatusOnlyArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceSecretCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceSecretsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceServiceAccountCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceServiceAccountsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceSubjectCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceSubjectsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NamespaceMetadata = {
    __typename?: 'NamespaceMetadata';
    annotations: Array<Label>;
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    creationTime?: Maybe<Scalars['Time']>;
    id: Scalars['ID'];
    labels: Array<Label>;
    name: Scalars['String'];
    priority: Scalars['Int'];
};

export type NetworkEntityInfo = {
    __typename?: 'NetworkEntityInfo';
    deployment?: Maybe<NetworkEntityInfo_Deployment>;
    desc?: Maybe<NetworkEntityInfoDesc>;
    externalSource?: Maybe<NetworkEntityInfo_ExternalSource>;
    id: Scalars['ID'];
    type: NetworkEntityInfo_Type;
};

export type NetworkEntityInfoDesc = NetworkEntityInfo_Deployment | NetworkEntityInfo_ExternalSource;

export type NetworkEntityInfo_Deployment = {
    __typename?: 'NetworkEntityInfo_Deployment';
    cluster: Scalars['String'];
    listenPorts: Array<Maybe<NetworkEntityInfo_Deployment_ListenPort>>;
    name: Scalars['String'];
    namespace: Scalars['String'];
};

export type NetworkEntityInfo_Deployment_ListenPort = {
    __typename?: 'NetworkEntityInfo_Deployment_ListenPort';
    l4Protocol: L4Protocol;
    port: Scalars['Int'];
};

export type NetworkEntityInfo_ExternalSource = {
    __typename?: 'NetworkEntityInfo_ExternalSource';
    default: Scalars['Boolean'];
    name: Scalars['String'];
};

export enum NetworkEntityInfo_Type {
    Deployment = 'DEPLOYMENT',
    ExternalSource = 'EXTERNAL_SOURCE',
    Internet = 'INTERNET',
    ListenEndpoint = 'LISTEN_ENDPOINT',
    UnknownType = 'UNKNOWN_TYPE',
}

export type NetworkFlow = {
    __typename?: 'NetworkFlow';
    clusterId: Scalars['String'];
    lastSeenTimestamp?: Maybe<Scalars['Time']>;
    props?: Maybe<NetworkFlowProperties>;
};

export type NetworkFlowProperties = {
    __typename?: 'NetworkFlowProperties';
    dstEntity?: Maybe<NetworkEntityInfo>;
    dstPort: Scalars['Int'];
    l4Protocol: L4Protocol;
    srcEntity?: Maybe<NetworkEntityInfo>;
};

export type Node = {
    __typename?: 'Node';
    annotations: Array<Label>;
    cluster: Cluster;
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    complianceResults: Array<ControlResult>;
    containerRuntime?: Maybe<ContainerRuntimeInfo>;
    containerRuntimeVersion: Scalars['String'];
    controlStatus: Scalars['String'];
    controls: Array<ComplianceControl>;
    externalIpAddresses: Array<Scalars['String']>;
    failingControls: Array<ComplianceControl>;
    id: Scalars['ID'];
    internalIpAddresses: Array<Scalars['String']>;
    joinedAt?: Maybe<Scalars['Time']>;
    k8SUpdated?: Maybe<Scalars['Time']>;
    kernelVersion: Scalars['String'];
    kubeProxyVersion: Scalars['String'];
    kubeletVersion: Scalars['String'];
    labels: Array<Label>;
    lastUpdated?: Maybe<Scalars['Time']>;
    name: Scalars['String'];
    nodeComplianceControlCount: ComplianceControlCount;
    nodeComponentCount: Scalars['Int'];
    nodeComponents: Array<NodeComponent>;
    nodeStatus: Scalars['String'];
    nodeVulnerabilities: Array<Maybe<NodeVulnerability>>;
    nodeVulnerabilityCount: Scalars['Int'];
    nodeVulnerabilityCounter: VulnerabilityCounter;
    notes: Array<Node_Note>;
    operatingSystem: Scalars['String'];
    osImage: Scalars['String'];
    passingControls: Array<ComplianceControl>;
    plottedNodeVulnerabilities: PlottedNodeVulnerabilities;
    priority: Scalars['Int'];
    riskScore: Scalars['Float'];
    scan?: Maybe<NodeScan>;
    scanNotes: Array<NodeScan_Note>;
    scanTime?: Maybe<Scalars['Time']>;
    taints: Array<Maybe<Taint>>;
    topNodeVulnerability?: Maybe<NodeVulnerability>;
    unusedVarSink?: Maybe<Scalars['Int']>;
};

export type NodeComplianceResultsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeControlStatusArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeControlsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeFailingControlsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeNodeComplianceControlCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeNodeComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeNodeComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type NodeNodeStatusArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeNodeVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type NodeNodeVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeNodeVulnerabilityCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodePassingControlsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodePlottedNodeVulnerabilitiesArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeTopNodeVulnerabilityArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeCve = {
    __typename?: 'NodeCVE';
    cveBaseInfo?: Maybe<CveInfo>;
    cvss: Scalars['Float'];
    id: Scalars['ID'];
    impactScore: Scalars['Float'];
    operatingSystem: Scalars['String'];
    severity: VulnerabilitySeverity;
    snoozeExpiry?: Maybe<Scalars['Time']>;
    snoozeStart?: Maybe<Scalars['Time']>;
    snoozed: Scalars['Boolean'];
};

export type NodeComponent = {
    __typename?: 'NodeComponent';
    fixedIn: Scalars['String'];
    id: Scalars['ID'];
    lastScanned?: Maybe<Scalars['Time']>;
    location: Scalars['String'];
    name: Scalars['String'];
    nodeCount: Scalars['Int'];
    nodeVulnerabilities: Array<Maybe<NodeVulnerability>>;
    nodeVulnerabilityCount: Scalars['Int'];
    nodeVulnerabilityCounter: VulnerabilityCounter;
    nodes: Array<Node>;
    operatingSystem: Scalars['String'];
    plottedNodeVulnerabilities: PlottedNodeVulnerabilities;
    priority: Scalars['Int'];
    riskScore: Scalars['Float'];
    source: Scalars['String'];
    topNodeVulnerability?: Maybe<NodeVulnerability>;
    unusedVarSink?: Maybe<Scalars['Int']>;
    version: Scalars['String'];
};

export type NodeComponentLocationArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeComponentNodeCountArgs = {
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type NodeComponentNodeVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type NodeComponentNodeVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type NodeComponentNodeVulnerabilityCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeComponentNodesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type NodeComponentPlottedNodeVulnerabilitiesArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeComponentUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeScan = {
    __typename?: 'NodeScan';
    /** @deprecated use 'nodeComponentCount' */
    componentCount: Scalars['Int'];
    /** @deprecated use 'nodeComponents' */
    components: Array<EmbeddedNodeScanComponent>;
    nodeComponentCount: Scalars['Int'];
    nodeComponents: Array<NodeComponent>;
    notes: Array<NodeScan_Note>;
    operatingSystem: Scalars['String'];
    scanTime?: Maybe<Scalars['Time']>;
};

export type NodeScanComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeScanComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type NodeScanNodeComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeScanNodeComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export enum NodeScan_Note {
    CertifiedRhelCvesUnavailable = 'CERTIFIED_RHEL_CVES_UNAVAILABLE',
    KernelUnsupported = 'KERNEL_UNSUPPORTED',
    Unset = 'UNSET',
    Unsupported = 'UNSUPPORTED',
}

export type NodeVulnerability = {
    __typename?: 'NodeVulnerability';
    createdAt?: Maybe<Scalars['Time']>;
    cve: Scalars['String'];
    cveBaseInfo?: Maybe<CveInfo>;
    cvss: Scalars['Float'];
    envImpact: Scalars['Float'];
    fixedByVersion: Scalars['String'];
    id: Scalars['ID'];
    impactScore: Scalars['Float'];
    isFixable: Scalars['Boolean'];
    lastModified?: Maybe<Scalars['Time']>;
    lastScanned?: Maybe<Scalars['Time']>;
    link: Scalars['String'];
    nodeComponentCount: Scalars['Int'];
    nodeComponents: Array<NodeComponent>;
    nodeCount: Scalars['Int'];
    nodes: Array<Node>;
    operatingSystem: Scalars['String'];
    publishedOn?: Maybe<Scalars['Time']>;
    scoreVersion: Scalars['String'];
    severity: Scalars['String'];
    summary: Scalars['String'];
    suppressActivation?: Maybe<Scalars['Time']>;
    suppressExpiry?: Maybe<Scalars['Time']>;
    suppressed: Scalars['Boolean'];
    unusedVarSink?: Maybe<Scalars['Int']>;
    vectors?: Maybe<EmbeddedVulnerabilityVectors>;
};

export type NodeVulnerabilityIsFixableArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeVulnerabilityNodeComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeVulnerabilityNodeComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type NodeVulnerabilityNodeCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type NodeVulnerabilityNodesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type NodeVulnerabilityUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export enum Node_Note {
    MissingScanData = 'MISSING_SCAN_DATA',
}

export type Notifier = {
    __typename?: 'Notifier';
    awsSecurityHub?: Maybe<AwsSecurityHub>;
    config?: Maybe<NotifierConfig>;
    cscc?: Maybe<Cscc>;
    email?: Maybe<Email>;
    generic?: Maybe<Generic>;
    id: Scalars['ID'];
    jira?: Maybe<Jira>;
    labelDefault: Scalars['String'];
    labelKey: Scalars['String'];
    name: Scalars['String'];
    pagerduty?: Maybe<PagerDuty>;
    splunk?: Maybe<Splunk>;
    sumologic?: Maybe<SumoLogic>;
    syslog?: Maybe<Syslog>;
    traits?: Maybe<Traits>;
    type: Scalars['String'];
    uiEndpoint: Scalars['String'];
};

export type NotifierConfig =
    | AwsSecurityHub
    | Cscc
    | Email
    | Generic
    | Jira
    | PagerDuty
    | Splunk
    | SumoLogic
    | Syslog;

export type OrchestratorMetadata = {
    __typename?: 'OrchestratorMetadata';
    apiVersions: Array<Scalars['String']>;
    buildDate?: Maybe<Scalars['Time']>;
    openshiftVersion: Scalars['String'];
    version: Scalars['String'];
};

export type PagerDuty = {
    __typename?: 'PagerDuty';
    apiKey: Scalars['String'];
};

export type Pagination = {
    limit?: InputMaybe<Scalars['Int']>;
    offset?: InputMaybe<Scalars['Int']>;
    sortOption?: InputMaybe<SortOption>;
    sortOptions?: InputMaybe<Array<InputMaybe<SortOption>>>;
};

export enum PermissionLevel {
    ClusterAdmin = 'CLUSTER_ADMIN',
    Default = 'DEFAULT',
    ElevatedClusterWide = 'ELEVATED_CLUSTER_WIDE',
    ElevatedInNamespace = 'ELEVATED_IN_NAMESPACE',
    None = 'NONE',
    Unset = 'UNSET',
}

export type PermissionSet = {
    __typename?: 'PermissionSet';
    description: Scalars['String'];
    id: Scalars['ID'];
    name: Scalars['String'];
    resourceToAccess: Array<Label>;
    traits?: Maybe<Traits>;
};

export type PlottedImageVulnerabilities = {
    __typename?: 'PlottedImageVulnerabilities';
    basicImageVulnerabilityCounter: VulnerabilityCounter;
    imageVulnerabilities: Array<Maybe<ImageVulnerability>>;
};

export type PlottedImageVulnerabilitiesImageVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
};

export type PlottedNodeVulnerabilities = {
    __typename?: 'PlottedNodeVulnerabilities';
    basicNodeVulnerabilityCounter: VulnerabilityCounter;
    nodeVulnerabilities: Array<Maybe<NodeVulnerability>>;
};

export type PlottedNodeVulnerabilitiesNodeVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
};

export type Pod = {
    __typename?: 'Pod';
    clusterId: Scalars['String'];
    containerCount: Scalars['Int'];
    deploymentId: Scalars['String'];
    events: Array<DeploymentEvent>;
    id: Scalars['ID'];
    liveInstances: Array<Maybe<ContainerInstance>>;
    name: Scalars['String'];
    namespace: Scalars['String'];
    started?: Maybe<Scalars['Time']>;
    terminatedInstances: Array<Maybe<Pod_ContainerInstanceList>>;
};

export type Pod_ContainerInstanceList = {
    __typename?: 'Pod_ContainerInstanceList';
    instances: Array<Maybe<ContainerInstance>>;
};

export type Policy = {
    __typename?: 'Policy';
    alertCount: Scalars['Int'];
    alerts: Array<Alert>;
    categories: Array<Scalars['String']>;
    criteriaLocked: Scalars['Boolean'];
    deploymentCount: Scalars['Int'];
    deployments: Array<Deployment>;
    description: Scalars['String'];
    disabled: Scalars['Boolean'];
    enforcementActions: Array<EnforcementAction>;
    eventSource: EventSource;
    exclusions: Array<Maybe<Exclusion>>;
    failingDeploymentCount: Scalars['Int'];
    failingDeployments: Array<Deployment>;
    fullMitreAttackVectors: Array<MitreAttackVector>;
    id: Scalars['ID'];
    isDefault: Scalars['Boolean'];
    lastUpdated?: Maybe<Scalars['Time']>;
    latestViolation?: Maybe<Scalars['Time']>;
    lifecycleStages: Array<LifecycleStage>;
    mitreAttackVectors: Array<Maybe<Policy_MitreAttackVectors>>;
    mitreVectorsLocked: Scalars['Boolean'];
    name: Scalars['String'];
    notifiers: Array<Scalars['String']>;
    policySections: Array<Maybe<PolicySection>>;
    policyStatus: Scalars['String'];
    policyVersion: Scalars['String'];
    rationale: Scalars['String'];
    remediation: Scalars['String'];
    sORTEnforcement: Scalars['Boolean'];
    sORTLifecycleStage: Scalars['String'];
    sORTName: Scalars['String'];
    scope: Array<Maybe<Scope>>;
    severity: Severity;
    unusedVarSink?: Maybe<Scalars['Int']>;
};

export type PolicyAlertCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type PolicyAlertsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type PolicyDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type PolicyDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type PolicyFailingDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type PolicyFailingDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type PolicyLatestViolationArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type PolicyPolicyStatusArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type PolicyUnusedVarSinkArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type PolicyCounter = {
    __typename?: 'PolicyCounter';
    critical: Scalars['Int'];
    high: Scalars['Int'];
    low: Scalars['Int'];
    medium: Scalars['Int'];
    total: Scalars['Int'];
};

export type PolicyGroup = {
    __typename?: 'PolicyGroup';
    booleanOperator: BooleanOperator;
    fieldName: Scalars['String'];
    negate: Scalars['Boolean'];
    values: Array<Maybe<PolicyValue>>;
};

export type PolicyRule = {
    __typename?: 'PolicyRule';
    apiGroups: Array<Scalars['String']>;
    nonResourceUrls: Array<Scalars['String']>;
    resourceNames: Array<Scalars['String']>;
    resources: Array<Scalars['String']>;
    verbs: Array<Scalars['String']>;
};

export type PolicySection = {
    __typename?: 'PolicySection';
    policyGroups: Array<Maybe<PolicyGroup>>;
    sectionName: Scalars['String'];
};

export type PolicyStatus = {
    __typename?: 'PolicyStatus';
    failingPolicies: Array<Policy>;
    status: Scalars['String'];
};

export type PolicyValue = {
    __typename?: 'PolicyValue';
    value: Scalars['String'];
};

export type PolicyViolationEvent = DeploymentEvent & {
    __typename?: 'PolicyViolationEvent';
    id: Scalars['ID'];
    name: Scalars['String'];
    timestamp?: Maybe<Scalars['Time']>;
};

export type Policy_MitreAttackVectors = {
    __typename?: 'Policy_MitreAttackVectors';
    tactic: Scalars['String'];
    techniques: Array<Scalars['String']>;
};

export type PortConfig = {
    __typename?: 'PortConfig';
    containerPort: Scalars['Int'];
    exposedPort: Scalars['Int'];
    exposure: PortConfig_ExposureLevel;
    exposureInfos: Array<Maybe<PortConfig_ExposureInfo>>;
    name: Scalars['String'];
    protocol: Scalars['String'];
};

export type PortConfig_ExposureInfo = {
    __typename?: 'PortConfig_ExposureInfo';
    externalHostnames: Array<Scalars['String']>;
    externalIps: Array<Scalars['String']>;
    level: PortConfig_ExposureLevel;
    nodePort: Scalars['Int'];
    serviceClusterIp: Scalars['String'];
    serviceId: Scalars['String'];
    serviceName: Scalars['String'];
    servicePort: Scalars['Int'];
};

export enum PortConfig_ExposureLevel {
    External = 'EXTERNAL',
    Host = 'HOST',
    Internal = 'INTERNAL',
    Node = 'NODE',
    Route = 'ROUTE',
    Unset = 'UNSET',
}

export type ProcessActivityEvent = DeploymentEvent & {
    __typename?: 'ProcessActivityEvent';
    args: Scalars['String'];
    id: Scalars['ID'];
    inBaseline: Scalars['Boolean'];
    name: Scalars['String'];
    parentName?: Maybe<Scalars['String']>;
    parentUid: Scalars['Int'];
    timestamp?: Maybe<Scalars['Time']>;
    uid: Scalars['Int'];
};

export type ProcessGroup = {
    __typename?: 'ProcessGroup';
    args: Scalars['String'];
    signals: Array<Maybe<ProcessIndicator>>;
};

export type ProcessIndicator = {
    __typename?: 'ProcessIndicator';
    clusterId: Scalars['String'];
    containerName: Scalars['String'];
    containerStartTime?: Maybe<Scalars['Time']>;
    deploymentId: Scalars['String'];
    id: Scalars['ID'];
    imageId: Scalars['String'];
    namespace: Scalars['String'];
    podId: Scalars['String'];
    podUid: Scalars['String'];
    signal?: Maybe<ProcessSignal>;
};

export type ProcessNameGroup = {
    __typename?: 'ProcessNameGroup';
    groups: Array<Maybe<ProcessGroup>>;
    name: Scalars['String'];
    timesExecuted: Scalars['Int'];
};

export type ProcessNoteKey = {
    args: Scalars['String'];
    containerName: Scalars['String'];
    deploymentID: Scalars['String'];
    execFilePath: Scalars['String'];
};

export type ProcessSignal = {
    __typename?: 'ProcessSignal';
    args: Scalars['String'];
    containerId: Scalars['String'];
    execFilePath: Scalars['String'];
    gid: Scalars['Int'];
    id: Scalars['ID'];
    lineage: Array<Scalars['String']>;
    lineageInfo: Array<Maybe<ProcessSignal_LineageInfo>>;
    name: Scalars['String'];
    pid: Scalars['Int'];
    scraped: Scalars['Boolean'];
    time?: Maybe<Scalars['Time']>;
    uid: Scalars['Int'];
};

export type ProcessSignal_LineageInfo = {
    __typename?: 'ProcessSignal_LineageInfo';
    parentExecFilePath: Scalars['String'];
    parentUid: Scalars['Int'];
};

export type ProviderMetadata = {
    __typename?: 'ProviderMetadata';
    aws?: Maybe<AwsProviderMetadata>;
    azure?: Maybe<AzureProviderMetadata>;
    google?: Maybe<GoogleProviderMetadata>;
    provider?: Maybe<ProviderMetadataProvider>;
    region: Scalars['String'];
    verified: Scalars['Boolean'];
    zone: Scalars['String'];
};

export type ProviderMetadataProvider =
    | AwsProviderMetadata
    | AzureProviderMetadata
    | GoogleProviderMetadata;

export type Query = {
    __typename?: 'Query';
    aggregatedResults: ComplianceAggregation_Response;
    cluster?: Maybe<Cluster>;
    clusterCount: Scalars['Int'];
    clusterHealthCounter: ClusterHealthCounter;
    clusterVulnerabilities: Array<ClusterVulnerability>;
    clusterVulnerability?: Maybe<ClusterVulnerability>;
    clusterVulnerabilityCount: Scalars['Int'];
    clusters: Array<Cluster>;
    complianceClusterCount: Scalars['Int'];
    complianceControl?: Maybe<ComplianceControl>;
    complianceControlGroup?: Maybe<ComplianceControlGroup>;
    complianceDeploymentCount: Scalars['Int'];
    complianceNamespaceCount: Scalars['Int'];
    complianceNodeCount: Scalars['Int'];
    complianceRecentRuns: Array<ComplianceRun>;
    complianceRun?: Maybe<ComplianceRun>;
    complianceRunStatuses: GetComplianceRunStatusesResponse;
    complianceStandard?: Maybe<ComplianceStandardMetadata>;
    complianceStandards: Array<ComplianceStandardMetadata>;
    deployment?: Maybe<Deployment>;
    deploymentCount: Scalars['Int'];
    deployments: Array<Deployment>;
    deploymentsWithMostSevereViolations: Array<DeploymentsWithMostSevereViolations>;
    executedControlCount: Scalars['Int'];
    executedControls: Array<ComplianceControlWithControlStatus>;
    fullImage?: Maybe<Image>;
    globalSearch: Array<SearchResult>;
    group?: Maybe<Group>;
    groupedContainerInstances: Array<ContainerNameGroup>;
    groups: Array<Group>;
    image?: Maybe<Image>;
    imageCVE?: Maybe<ImageCveCore>;
    imageCVECount: Scalars['Int'];
    imageCVEs: Array<ImageCveCore>;
    imageComponent?: Maybe<ImageComponent>;
    imageComponentCount: Scalars['Int'];
    imageComponents: Array<ImageComponent>;
    imageCount: Scalars['Int'];
    imageVulnerabilities: Array<ImageVulnerability>;
    imageVulnerability?: Maybe<ImageVulnerability>;
    imageVulnerabilityCount: Scalars['Int'];
    images: Array<Image>;
    istioClusterVulnerabilities: Array<ClusterVulnerability>;
    istioClusterVulnerability?: Maybe<ClusterVulnerability>;
    istioClusterVulnerabilityCount: Scalars['Int'];
    k8sClusterVulnerabilities: Array<ClusterVulnerability>;
    k8sClusterVulnerability?: Maybe<ClusterVulnerability>;
    k8sClusterVulnerabilityCount: Scalars['Int'];
    k8sRole?: Maybe<K8SRole>;
    k8sRoleCount: Scalars['Int'];
    k8sRoles: Array<K8SRole>;
    myPermissions?: Maybe<GetPermissionsResponse>;
    namespace?: Maybe<Namespace>;
    namespaceByClusterIDAndName?: Maybe<Namespace>;
    namespaceCount: Scalars['Int'];
    namespaces: Array<Namespace>;
    node?: Maybe<Node>;
    nodeComponent?: Maybe<NodeComponent>;
    nodeComponentCount: Scalars['Int'];
    nodeComponents: Array<NodeComponent>;
    nodeCount: Scalars['Int'];
    nodeVulnerabilities: Array<NodeVulnerability>;
    nodeVulnerability?: Maybe<NodeVulnerability>;
    nodeVulnerabilityCount: Scalars['Int'];
    nodes: Array<Node>;
    notifier?: Maybe<Notifier>;
    notifiers: Array<Notifier>;
    openShiftClusterVulnerabilities: Array<ClusterVulnerability>;
    openShiftClusterVulnerability?: Maybe<ClusterVulnerability>;
    openShiftClusterVulnerabilityCount: Scalars['Int'];
    permissionSet?: Maybe<PermissionSet>;
    permissionSets: Array<PermissionSet>;
    pod?: Maybe<Pod>;
    podCount: Scalars['Int'];
    pods: Array<Pod>;
    policies: Array<Policy>;
    policy?: Maybe<Policy>;
    policyCount: Scalars['Int'];
    role?: Maybe<Role>;
    roles: Array<Role>;
    searchAutocomplete: Array<Scalars['String']>;
    searchOptions: Array<Scalars['String']>;
    secret?: Maybe<Secret>;
    secretCount: Scalars['Int'];
    secrets: Array<Secret>;
    serviceAccount?: Maybe<ServiceAccount>;
    serviceAccountCount: Scalars['Int'];
    serviceAccounts: Array<ServiceAccount>;
    simpleAccessScope?: Maybe<SimpleAccessScope>;
    simpleAccessScopes: Array<SimpleAccessScope>;
    subject?: Maybe<Subject>;
    subjectCount: Scalars['Int'];
    subjects: Array<Subject>;
    token?: Maybe<TokenMetadata>;
    tokens: Array<TokenMetadata>;
    violation?: Maybe<Alert>;
    violationCount: Scalars['Int'];
    violations: Array<Alert>;
    vulnerabilityRequest?: Maybe<VulnerabilityRequest>;
    vulnerabilityRequests: Array<VulnerabilityRequest>;
    vulnerabilityRequestsCount: Scalars['Int'];
};

export type QueryAggregatedResultsArgs = {
    collapseBy?: InputMaybe<ComplianceAggregation_Scope>;
    groupBy?: InputMaybe<Array<ComplianceAggregation_Scope>>;
    unit: ComplianceAggregation_Scope;
    where?: InputMaybe<Scalars['String']>;
};

export type QueryClusterArgs = {
    id: Scalars['ID'];
};

export type QueryClusterCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryClusterHealthCounterArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryClusterVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type QueryClusterVulnerabilityArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QueryClusterVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryClustersArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryComplianceClusterCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryComplianceControlArgs = {
    id: Scalars['ID'];
};

export type QueryComplianceControlGroupArgs = {
    id: Scalars['ID'];
};

export type QueryComplianceDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryComplianceNamespaceCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryComplianceNodeCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryComplianceRecentRunsArgs = {
    clusterId?: InputMaybe<Scalars['ID']>;
    since?: InputMaybe<Scalars['Time']>;
    standardId?: InputMaybe<Scalars['ID']>;
};

export type QueryComplianceRunArgs = {
    id: Scalars['ID'];
};

export type QueryComplianceRunStatusesArgs = {
    ids: Array<Scalars['ID']>;
};

export type QueryComplianceStandardArgs = {
    id: Scalars['ID'];
};

export type QueryComplianceStandardsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryDeploymentArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QueryDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryDeploymentsWithMostSevereViolationsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryExecutedControlCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryExecutedControlsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryFullImageArgs = {
    id: Scalars['ID'];
};

export type QueryGlobalSearchArgs = {
    categories?: InputMaybe<Array<SearchCategory>>;
    query: Scalars['String'];
};

export type QueryGroupArgs = {
    authProviderId?: InputMaybe<Scalars['String']>;
    id?: InputMaybe<Scalars['String']>;
    key?: InputMaybe<Scalars['String']>;
    value?: InputMaybe<Scalars['String']>;
};

export type QueryGroupedContainerInstancesArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryImageArgs = {
    id: Scalars['ID'];
};

export type QueryImageCveArgs = {
    cve?: InputMaybe<Scalars['String']>;
    subfieldScopeQuery?: InputMaybe<Scalars['String']>;
};

export type QueryImageCveCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryImageCvEsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryImageComponentArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QueryImageComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryImageComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type QueryImageCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryImageVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type QueryImageVulnerabilityArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QueryImageVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryImagesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryIstioClusterVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryIstioClusterVulnerabilityArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QueryIstioClusterVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryK8sClusterVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryK8sClusterVulnerabilityArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QueryK8sClusterVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryK8sRoleArgs = {
    id: Scalars['ID'];
};

export type QueryK8sRoleCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryK8sRolesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryNamespaceArgs = {
    id: Scalars['ID'];
};

export type QueryNamespaceByClusterIdAndNameArgs = {
    clusterID: Scalars['ID'];
    name: Scalars['String'];
};

export type QueryNamespaceCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryNamespacesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryNodeArgs = {
    id: Scalars['ID'];
};

export type QueryNodeComponentArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QueryNodeComponentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryNodeComponentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type QueryNodeCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryNodeVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    scopeQuery?: InputMaybe<Scalars['String']>;
};

export type QueryNodeVulnerabilityArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QueryNodeVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryNodesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryNotifierArgs = {
    id: Scalars['ID'];
};

export type QueryOpenShiftClusterVulnerabilitiesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryOpenShiftClusterVulnerabilityArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QueryOpenShiftClusterVulnerabilityCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryPermissionSetArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QueryPodArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QueryPodCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryPodsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryPoliciesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryPolicyArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QueryPolicyCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryRoleArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QuerySearchAutocompleteArgs = {
    categories?: InputMaybe<Array<SearchCategory>>;
    query: Scalars['String'];
};

export type QuerySearchOptionsArgs = {
    categories?: InputMaybe<Array<SearchCategory>>;
};

export type QuerySecretArgs = {
    id: Scalars['ID'];
};

export type QuerySecretCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QuerySecretsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryServiceAccountArgs = {
    id: Scalars['ID'];
};

export type QueryServiceAccountCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryServiceAccountsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QuerySimpleAccessScopeArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QuerySubjectArgs = {
    id?: InputMaybe<Scalars['ID']>;
};

export type QuerySubjectCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QuerySubjectsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryTokenArgs = {
    id: Scalars['ID'];
};

export type QueryTokensArgs = {
    revoked?: InputMaybe<Scalars['Boolean']>;
};

export type QueryViolationArgs = {
    id: Scalars['ID'];
};

export type QueryViolationCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type QueryViolationsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type QueryVulnerabilityRequestArgs = {
    id: Scalars['ID'];
};

export type QueryVulnerabilityRequestsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
    requestIDSelector?: InputMaybe<Scalars['String']>;
};

export type QueryVulnerabilityRequestsCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ReadinessProbe = {
    __typename?: 'ReadinessProbe';
    defined: Scalars['Boolean'];
};

export type RequestComment = {
    __typename?: 'RequestComment';
    createdAt?: Maybe<Scalars['Time']>;
    id: Scalars['ID'];
    message: Scalars['String'];
    user?: Maybe<SlimUser>;
};

export type Resource =
    | ComplianceDomain_Cluster
    | ComplianceDomain_Deployment
    | ComplianceDomain_Node;

export type ResourceCountByCveSeverity = {
    __typename?: 'ResourceCountByCVESeverity';
    critical: ResourceCountByFixability;
    important: ResourceCountByFixability;
    low: ResourceCountByFixability;
    moderate: ResourceCountByFixability;
};

export type ResourceCountByFixability = {
    __typename?: 'ResourceCountByFixability';
    fixable: Scalars['Int'];
    total: Scalars['Int'];
};

export type Resources = {
    __typename?: 'Resources';
    cpuCoresLimit: Scalars['Float'];
    cpuCoresRequest: Scalars['Float'];
    memoryMbLimit: Scalars['Float'];
    memoryMbRequest: Scalars['Float'];
};

export type Risk = {
    __typename?: 'Risk';
    id: Scalars['ID'];
    results: Array<Maybe<Risk_Result>>;
    score: Scalars['Float'];
    subject?: Maybe<RiskSubject>;
};

export type RiskSubject = {
    __typename?: 'RiskSubject';
    clusterId: Scalars['String'];
    id: Scalars['ID'];
    namespace: Scalars['String'];
    type: RiskSubjectType;
};

export enum RiskSubjectType {
    Cluster = 'CLUSTER',
    Deployment = 'DEPLOYMENT',
    Image = 'IMAGE',
    ImageComponent = 'IMAGE_COMPONENT',
    Namespace = 'NAMESPACE',
    Node = 'NODE',
    NodeComponent = 'NODE_COMPONENT',
    Serviceaccount = 'SERVICEACCOUNT',
    Unknown = 'UNKNOWN',
}

export type Risk_Result = {
    __typename?: 'Risk_Result';
    factors: Array<Maybe<Risk_Result_Factor>>;
    name: Scalars['String'];
    score: Scalars['Float'];
};

export type Risk_Result_Factor = {
    __typename?: 'Risk_Result_Factor';
    message: Scalars['String'];
    url: Scalars['String'];
};

export type Role = {
    __typename?: 'Role';
    accessScopeId: Scalars['String'];
    description: Scalars['String'];
    globalAccess: Access;
    name: Scalars['String'];
    permissionSetId: Scalars['String'];
    resourceToAccess: Array<Label>;
    traits?: Maybe<Traits>;
};

export type ScannerHealthInfo = {
    __typename?: 'ScannerHealthInfo';
    statusErrors: Array<Scalars['String']>;
};

export type Scope = {
    __typename?: 'Scope';
    cluster: Scalars['String'];
    label?: Maybe<Scope_Label>;
    namespace: Scalars['String'];
};

export type Scope_Label = {
    __typename?: 'Scope_Label';
    key: Scalars['String'];
    value: Scalars['String'];
};

export type ScopedPermissions = {
    __typename?: 'ScopedPermissions';
    permissions: Array<StringListEntry>;
    scope: Scalars['String'];
};

export enum SearchCategory {
    ActiveComponent = 'ACTIVE_COMPONENT',
    Alerts = 'ALERTS',
    ApiToken = 'API_TOKEN',
    Blob = 'BLOB',
    Clusters = 'CLUSTERS',
    ClusterHealth = 'CLUSTER_HEALTH',
    ClusterVulnerabilities = 'CLUSTER_VULNERABILITIES',
    ClusterVulnEdge = 'CLUSTER_VULN_EDGE',
    Collections = 'COLLECTIONS',
    Compliance = 'COMPLIANCE',
    ComplianceCheckResults = 'COMPLIANCE_CHECK_RESULTS',
    ComplianceControl = 'COMPLIANCE_CONTROL',
    ComplianceControlGroup = 'COMPLIANCE_CONTROL_GROUP',
    ComplianceDomain = 'COMPLIANCE_DOMAIN',
    ComplianceIntegrations = 'COMPLIANCE_INTEGRATIONS',
    ComplianceMetadata = 'COMPLIANCE_METADATA',
    ComplianceResults = 'COMPLIANCE_RESULTS',
    ComplianceScan = 'COMPLIANCE_SCAN',
    ComplianceScanSettings = 'COMPLIANCE_SCAN_SETTINGS',
    ComplianceStandard = 'COMPLIANCE_STANDARD',
    ComponentVulnEdge = 'COMPONENT_VULN_EDGE',
    Deployments = 'DEPLOYMENTS',
    Images = 'IMAGES',
    ImageComponents = 'IMAGE_COMPONENTS',
    ImageComponentEdge = 'IMAGE_COMPONENT_EDGE',
    ImageIntegrations = 'IMAGE_INTEGRATIONS',
    ImageVulnerabilities = 'IMAGE_VULNERABILITIES',
    ImageVulnEdge = 'IMAGE_VULN_EDGE',
    Namespaces = 'NAMESPACES',
    NetworkBaseline = 'NETWORK_BASELINE',
    NetworkEntity = 'NETWORK_ENTITY',
    NetworkPolicies = 'NETWORK_POLICIES',
    Nodes = 'NODES',
    NodeComponents = 'NODE_COMPONENTS',
    NodeComponentCveEdge = 'NODE_COMPONENT_CVE_EDGE',
    NodeComponentEdge = 'NODE_COMPONENT_EDGE',
    NodeVulnerabilities = 'NODE_VULNERABILITIES',
    NodeVulnEdge = 'NODE_VULN_EDGE',
    Pods = 'PODS',
    Policies = 'POLICIES',
    PolicyCategories = 'POLICY_CATEGORIES',
    PolicyCategoryEdge = 'POLICY_CATEGORY_EDGE',
    ProcessBaselines = 'PROCESS_BASELINES',
    ProcessBaselineResults = 'PROCESS_BASELINE_RESULTS',
    ProcessIndicators = 'PROCESS_INDICATORS',
    ProcessListeningOnPort = 'PROCESS_LISTENING_ON_PORT',
    ReportConfigurations = 'REPORT_CONFIGURATIONS',
    ReportMetadata = 'REPORT_METADATA',
    ReportSnapshot = 'REPORT_SNAPSHOT',
    Risks = 'RISKS',
    Rolebindings = 'ROLEBINDINGS',
    Roles = 'ROLES',
    SearchUnset = 'SEARCH_UNSET',
    Secrets = 'SECRETS',
    ServiceAccounts = 'SERVICE_ACCOUNTS',
    Subjects = 'SUBJECTS',
    Vulnerabilities = 'VULNERABILITIES',
    VulnRequest = 'VULN_REQUEST',
}

export type SearchResult = {
    __typename?: 'SearchResult';
    category: SearchCategory;
    id: Scalars['ID'];
    location: Scalars['String'];
    name: Scalars['String'];
    score: Scalars['Float'];
};

export type Secret = {
    __typename?: 'Secret';
    annotations: Array<Label>;
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    createdAt?: Maybe<Scalars['Time']>;
    deploymentCount: Scalars['Int'];
    deployments: Array<Deployment>;
    files: Array<Maybe<SecretDataFile>>;
    id: Scalars['ID'];
    labels: Array<Label>;
    name: Scalars['String'];
    namespace: Scalars['String'];
    relationship?: Maybe<SecretRelationship>;
    type: Scalars['String'];
};

export type SecretDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type SecretDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type SecretContainerRelationship = {
    __typename?: 'SecretContainerRelationship';
    id: Scalars['ID'];
    path: Scalars['String'];
};

export type SecretDataFile = {
    __typename?: 'SecretDataFile';
    cert?: Maybe<Cert>;
    imagePullSecret?: Maybe<ImagePullSecret>;
    metadata?: Maybe<SecretDataFileMetadata>;
    name: Scalars['String'];
    type: SecretType;
};

export type SecretDataFileMetadata = Cert | ImagePullSecret;

export type SecretDeploymentRelationship = {
    __typename?: 'SecretDeploymentRelationship';
    id: Scalars['ID'];
    name: Scalars['String'];
};

export type SecretRelationship = {
    __typename?: 'SecretRelationship';
    containerRelationships: Array<Maybe<SecretContainerRelationship>>;
    deploymentRelationships: Array<Maybe<SecretDeploymentRelationship>>;
    id: Scalars['ID'];
};

export enum SecretType {
    CertificateRequest = 'CERTIFICATE_REQUEST',
    CertPrivateKey = 'CERT_PRIVATE_KEY',
    DsaPrivateKey = 'DSA_PRIVATE_KEY',
    EcPrivateKey = 'EC_PRIVATE_KEY',
    EncryptedPrivateKey = 'ENCRYPTED_PRIVATE_KEY',
    ImagePullSecret = 'IMAGE_PULL_SECRET',
    OpensshPrivateKey = 'OPENSSH_PRIVATE_KEY',
    PgpPrivateKey = 'PGP_PRIVATE_KEY',
    PrivacyEnhancedMessage = 'PRIVACY_ENHANCED_MESSAGE',
    PublicCertificate = 'PUBLIC_CERTIFICATE',
    RsaPrivateKey = 'RSA_PRIVATE_KEY',
    Undetermined = 'UNDETERMINED',
}

export type SecurityContext = {
    __typename?: 'SecurityContext';
    addCapabilities: Array<Scalars['String']>;
    allowPrivilegeEscalation: Scalars['Boolean'];
    dropCapabilities: Array<Scalars['String']>;
    privileged: Scalars['Boolean'];
    readOnlyRootFilesystem: Scalars['Boolean'];
    seccompProfile?: Maybe<SecurityContext_SeccompProfile>;
    selinux?: Maybe<SecurityContext_SeLinux>;
};

export type SecurityContext_SeLinux = {
    __typename?: 'SecurityContext_SELinux';
    level: Scalars['String'];
    role: Scalars['String'];
    type: Scalars['String'];
    user: Scalars['String'];
};

export type SecurityContext_SeccompProfile = {
    __typename?: 'SecurityContext_SeccompProfile';
    localhostProfile: Scalars['String'];
    type: SecurityContext_SeccompProfile_ProfileType;
};

export enum SecurityContext_SeccompProfile_ProfileType {
    Localhost = 'LOCALHOST',
    RuntimeDefault = 'RUNTIME_DEFAULT',
    Unconfined = 'UNCONFINED',
}

export type SensorDeploymentIdentification = {
    __typename?: 'SensorDeploymentIdentification';
    appNamespace: Scalars['String'];
    appNamespaceId: Scalars['String'];
    appServiceaccountId: Scalars['String'];
    defaultNamespaceId: Scalars['String'];
    k8SNodeName: Scalars['String'];
    systemNamespaceId: Scalars['String'];
};

export type ServiceAccount = {
    __typename?: 'ServiceAccount';
    annotations: Array<Label>;
    automountToken: Scalars['Boolean'];
    cluster: Cluster;
    clusterAdmin: Scalars['Boolean'];
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    createdAt?: Maybe<Scalars['Time']>;
    deploymentCount: Scalars['Int'];
    deployments: Array<Deployment>;
    id: Scalars['ID'];
    imagePullSecretCount: Scalars['Int'];
    imagePullSecretObjects: Array<Secret>;
    imagePullSecrets: Array<Scalars['String']>;
    k8sRoleCount: Scalars['Int'];
    k8sRoles: Array<K8SRole>;
    labels: Array<Label>;
    name: Scalars['String'];
    namespace: Scalars['String'];
    saNamespace: Namespace;
    scopedPermissions: Array<ScopedPermissions>;
    secrets: Array<Scalars['String']>;
};

export type ServiceAccountDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ServiceAccountDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type ServiceAccountImagePullSecretObjectsArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ServiceAccountK8sRoleCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type ServiceAccountK8sRolesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type SetBasedLabelSelector = {
    __typename?: 'SetBasedLabelSelector';
    requirements: Array<Maybe<SetBasedLabelSelector_Requirement>>;
};

export enum SetBasedLabelSelector_Operator {
    Exists = 'EXISTS',
    In = 'IN',
    NotExists = 'NOT_EXISTS',
    NotIn = 'NOT_IN',
    Unknown = 'UNKNOWN',
}

export type SetBasedLabelSelector_Requirement = {
    __typename?: 'SetBasedLabelSelector_Requirement';
    key: Scalars['String'];
    op: SetBasedLabelSelector_Operator;
    values: Array<Scalars['String']>;
};

export enum Severity {
    CriticalSeverity = 'CRITICAL_SEVERITY',
    HighSeverity = 'HIGH_SEVERITY',
    LowSeverity = 'LOW_SEVERITY',
    MediumSeverity = 'MEDIUM_SEVERITY',
    UnsetSeverity = 'UNSET_SEVERITY',
}

export type Signature = {
    __typename?: 'Signature';
    cosign?: Maybe<CosignSignature>;
    signature?: Maybe<SignatureSignature>;
};

export type SignatureSignature = CosignSignature;

export type SimpleAccessScope = {
    __typename?: 'SimpleAccessScope';
    description: Scalars['String'];
    id: Scalars['ID'];
    name: Scalars['String'];
    rules?: Maybe<SimpleAccessScope_Rules>;
    traits?: Maybe<Traits>;
};

export type SimpleAccessScope_Rules = {
    __typename?: 'SimpleAccessScope_Rules';
    clusterLabelSelectors: Array<Maybe<SetBasedLabelSelector>>;
    includedClusters: Array<Scalars['String']>;
    includedNamespaces: Array<Maybe<SimpleAccessScope_Rules_Namespace>>;
    namespaceLabelSelectors: Array<Maybe<SetBasedLabelSelector>>;
};

export type SimpleAccessScope_Rules_Namespace = {
    __typename?: 'SimpleAccessScope_Rules_Namespace';
    clusterName: Scalars['String'];
    namespaceName: Scalars['String'];
};

export type SlimUser = {
    __typename?: 'SlimUser';
    id: Scalars['ID'];
    name: Scalars['String'];
};

export type SortOption = {
    aggregateBy?: InputMaybe<AggregateBy>;
    field?: InputMaybe<Scalars['String']>;
    reversed?: InputMaybe<Scalars['Boolean']>;
};

export enum SourceType {
    Dotnetcoreruntime = 'DOTNETCORERUNTIME',
    Infrastructure = 'INFRASTRUCTURE',
    Java = 'JAVA',
    Nodejs = 'NODEJS',
    Os = 'OS',
    Python = 'PYTHON',
    Ruby = 'RUBY',
}

export type Splunk = {
    __typename?: 'Splunk';
    auditLoggingEnabled: Scalars['Boolean'];
    httpEndpoint: Scalars['String'];
    httpToken: Scalars['String'];
    insecure: Scalars['Boolean'];
    sourceTypes: Array<Label>;
    truncate: Scalars['Int'];
};

export type StaticClusterConfig = {
    __typename?: 'StaticClusterConfig';
    admissionController: Scalars['Boolean'];
    admissionControllerEvents: Scalars['Boolean'];
    admissionControllerUpdates: Scalars['Boolean'];
    centralApiEndpoint: Scalars['String'];
    collectionMethod: CollectionMethod;
    collectorImage: Scalars['String'];
    mainImage: Scalars['String'];
    slimCollector: Scalars['Boolean'];
    tolerationsConfig?: Maybe<TolerationsConfig>;
    type: ClusterType;
};

export type StringListEntry = {
    __typename?: 'StringListEntry';
    key: Scalars['String'];
    values: Array<Scalars['String']>;
};

export type Subject = {
    __typename?: 'Subject';
    clusterAdmin: Scalars['Boolean'];
    clusterId: Scalars['String'];
    clusterName: Scalars['String'];
    id: Scalars['ID'];
    k8sRoleCount: Scalars['Int'];
    k8sRoles: Array<K8SRole>;
    kind: SubjectKind;
    name: Scalars['String'];
    namespace: Scalars['String'];
    scopedPermissions: Array<ScopedPermissions>;
    type: Scalars['String'];
};

export type SubjectK8sRoleCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type SubjectK8sRolesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export enum SubjectKind {
    Group = 'GROUP',
    ServiceAccount = 'SERVICE_ACCOUNT',
    UnsetKind = 'UNSET_KIND',
    User = 'USER',
}

export type SumoLogic = {
    __typename?: 'SumoLogic';
    httpSourceAddress: Scalars['String'];
    skipTLSVerify: Scalars['Boolean'];
};

export type Syslog = {
    __typename?: 'Syslog';
    endpoint?: Maybe<SyslogEndpoint>;
    extraFields: Array<Maybe<KeyValuePair>>;
    localFacility: Syslog_LocalFacility;
    messageFormat: Syslog_MessageFormat;
    tcpConfig?: Maybe<Syslog_TcpConfig>;
};

export type SyslogEndpoint = Syslog_TcpConfig;

export enum Syslog_LocalFacility {
    Local0 = 'LOCAL0',
    Local1 = 'LOCAL1',
    Local2 = 'LOCAL2',
    Local3 = 'LOCAL3',
    Local4 = 'LOCAL4',
    Local5 = 'LOCAL5',
    Local6 = 'LOCAL6',
    Local7 = 'LOCAL7',
}

export enum Syslog_MessageFormat {
    Cef = 'CEF',
    Legacy = 'LEGACY',
}

export type Syslog_TcpConfig = {
    __typename?: 'Syslog_TCPConfig';
    hostname: Scalars['String'];
    port: Scalars['Int'];
    skipTlsVerify: Scalars['Boolean'];
    useTls: Scalars['Boolean'];
};

export type Taint = {
    __typename?: 'Taint';
    key: Scalars['String'];
    taintEffect: TaintEffect;
    value: Scalars['String'];
};

export enum TaintEffect {
    NoExecuteTaintEffect = 'NO_EXECUTE_TAINT_EFFECT',
    NoScheduleTaintEffect = 'NO_SCHEDULE_TAINT_EFFECT',
    PreferNoScheduleTaintEffect = 'PREFER_NO_SCHEDULE_TAINT_EFFECT',
    UnknownTaintEffect = 'UNKNOWN_TAINT_EFFECT',
}

export type TokenMetadata = {
    __typename?: 'TokenMetadata';
    expiration?: Maybe<Scalars['Time']>;
    id: Scalars['ID'];
    issuedAt?: Maybe<Scalars['Time']>;
    name: Scalars['String'];
    revoked: Scalars['Boolean'];
    role: Scalars['String'];
    roles: Array<Scalars['String']>;
};

export type Toleration = {
    __typename?: 'Toleration';
    key: Scalars['String'];
    operator: Toleration_Operator;
    taintEffect: TaintEffect;
    value: Scalars['String'];
};

export enum Toleration_Operator {
    TolerationOperationUnknown = 'TOLERATION_OPERATION_UNKNOWN',
    TolerationOperatorEqual = 'TOLERATION_OPERATOR_EQUAL',
    TolerationOperatorExists = 'TOLERATION_OPERATOR_EXISTS',
}

export type TolerationsConfig = {
    __typename?: 'TolerationsConfig';
    disabled: Scalars['Boolean'];
};

export type Traits = {
    __typename?: 'Traits';
    mutabilityMode: Traits_MutabilityMode;
    origin: Traits_Origin;
    visibility: Traits_Visibility;
};

export enum Traits_MutabilityMode {
    AllowMutate = 'ALLOW_MUTATE',
    AllowMutateForced = 'ALLOW_MUTATE_FORCED',
}

export enum Traits_Origin {
    Declarative = 'DECLARATIVE',
    DeclarativeOrphaned = 'DECLARATIVE_ORPHANED',
    Default = 'DEFAULT',
    Imperative = 'IMPERATIVE',
}

export enum Traits_Visibility {
    Hidden = 'HIDDEN',
    Visible = 'VISIBLE',
}

export type UpgradeProgress = {
    __typename?: 'UpgradeProgress';
    since?: Maybe<Scalars['Time']>;
    upgradeState: UpgradeProgress_UpgradeState;
    upgradeStatusDetail: Scalars['String'];
};

export enum UpgradeProgress_UpgradeState {
    PreFlightChecksComplete = 'PRE_FLIGHT_CHECKS_COMPLETE',
    PreFlightChecksFailed = 'PRE_FLIGHT_CHECKS_FAILED',
    UpgraderLaunched = 'UPGRADER_LAUNCHED',
    UpgraderLaunching = 'UPGRADER_LAUNCHING',
    UpgradeComplete = 'UPGRADE_COMPLETE',
    UpgradeErrorRollbackFailed = 'UPGRADE_ERROR_ROLLBACK_FAILED',
    UpgradeErrorRolledBack = 'UPGRADE_ERROR_ROLLED_BACK',
    UpgradeErrorRollingBack = 'UPGRADE_ERROR_ROLLING_BACK',
    UpgradeErrorUnknown = 'UPGRADE_ERROR_UNKNOWN',
    UpgradeInitializationError = 'UPGRADE_INITIALIZATION_ERROR',
    UpgradeInitializing = 'UPGRADE_INITIALIZING',
    UpgradeOperationsDone = 'UPGRADE_OPERATIONS_DONE',
    UpgradeTimedOut = 'UPGRADE_TIMED_OUT',
}

export type V1Metadata = {
    __typename?: 'V1Metadata';
    author: Scalars['String'];
    command: Array<Scalars['String']>;
    created?: Maybe<Scalars['Time']>;
    digest: Scalars['String'];
    entrypoint: Array<Scalars['String']>;
    labels: Array<Label>;
    layers: Array<Maybe<ImageLayer>>;
    user: Scalars['String'];
    volumes: Array<Scalars['String']>;
};

export type V2Metadata = {
    __typename?: 'V2Metadata';
    digest: Scalars['String'];
};

export enum ViolationState {
    Active = 'ACTIVE',
    Attempted = 'ATTEMPTED',
    Resolved = 'RESOLVED',
    Snoozed = 'SNOOZED',
}

export type Volume = {
    __typename?: 'Volume';
    destination: Scalars['String'];
    mountPropagation: Volume_MountPropagation;
    name: Scalars['String'];
    readOnly: Scalars['Boolean'];
    source: Scalars['String'];
    type: Scalars['String'];
};

export enum Volume_MountPropagation {
    Bidirectional = 'BIDIRECTIONAL',
    HostToContainer = 'HOST_TO_CONTAINER',
    None = 'NONE',
}

export type VulnReqExpiry = {
    expiresOn?: InputMaybe<Scalars['Time']>;
    expiresWhenFixed?: InputMaybe<Scalars['Boolean']>;
};

export type VulnReqGlobalScope = {
    images?: InputMaybe<VulnReqImageScope>;
};

export type VulnReqImageScope = {
    registry?: InputMaybe<Scalars['String']>;
    remote?: InputMaybe<Scalars['String']>;
    tag?: InputMaybe<Scalars['String']>;
};

export type VulnReqScope = {
    globalScope?: InputMaybe<VulnReqGlobalScope>;
    imageScope?: InputMaybe<VulnReqImageScope>;
};

export type VulnerabilityCounter = {
    __typename?: 'VulnerabilityCounter';
    all: VulnerabilityFixableCounterResolver;
    critical: VulnerabilityFixableCounterResolver;
    important: VulnerabilityFixableCounterResolver;
    low: VulnerabilityFixableCounterResolver;
    moderate: VulnerabilityFixableCounterResolver;
};

export type VulnerabilityFixableCounterResolver = {
    __typename?: 'VulnerabilityFixableCounterResolver';
    fixable: Scalars['Int'];
    total: Scalars['Int'];
};

export type VulnerabilityRequest = {
    __typename?: 'VulnerabilityRequest';
    LastUpdated?: Maybe<Scalars['Time']>;
    approvers: Array<SlimUser>;
    comments: Array<RequestComment>;
    createdAt?: Maybe<Scalars['Time']>;
    cves?: Maybe<VulnerabilityRequest_CvEs>;
    deferralReq?: Maybe<DeferralRequest>;
    deploymentCount: Scalars['Int'];
    deployments: Array<Deployment>;
    expired: Scalars['Boolean'];
    falsePositiveReq?: Maybe<FalsePositiveRequest>;
    id: Scalars['ID'];
    imageCount: Scalars['Int'];
    images: Array<Image>;
    requestor?: Maybe<SlimUser>;
    scope?: Maybe<VulnerabilityRequest_Scope>;
    status: Scalars['String'];
    targetState: Scalars['String'];
    updatedDeferralReq?: Maybe<DeferralRequest>;
};

export type VulnerabilityRequestDeploymentCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type VulnerabilityRequestDeploymentsArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type VulnerabilityRequestImageCountArgs = {
    query?: InputMaybe<Scalars['String']>;
};

export type VulnerabilityRequestImagesArgs = {
    pagination?: InputMaybe<Pagination>;
    query?: InputMaybe<Scalars['String']>;
};

export type VulnerabilityRequest_CvEs = {
    __typename?: 'VulnerabilityRequest_CVEs';
    cves: Array<Scalars['String']>;
};

export type VulnerabilityRequest_Scope = {
    __typename?: 'VulnerabilityRequest_Scope';
    globalScope?: Maybe<VulnerabilityRequest_Scope_Global>;
    imageScope?: Maybe<VulnerabilityRequest_Scope_Image>;
    info?: Maybe<VulnerabilityRequest_ScopeInfo>;
};

export type VulnerabilityRequest_ScopeInfo =
    | VulnerabilityRequest_Scope_Global
    | VulnerabilityRequest_Scope_Image;

export type VulnerabilityRequest_Scope_Global = {
    __typename?: 'VulnerabilityRequest_Scope_Global';
};

export type VulnerabilityRequest_Scope_Image = {
    __typename?: 'VulnerabilityRequest_Scope_Image';
    registry: Scalars['String'];
    remote: Scalars['String'];
    tag: Scalars['String'];
};

export enum VulnerabilitySeverity {
    CriticalVulnerabilitySeverity = 'CRITICAL_VULNERABILITY_SEVERITY',
    ImportantVulnerabilitySeverity = 'IMPORTANT_VULNERABILITY_SEVERITY',
    LowVulnerabilitySeverity = 'LOW_VULNERABILITY_SEVERITY',
    ModerateVulnerabilitySeverity = 'MODERATE_VULNERABILITY_SEVERITY',
    UnknownVulnerabilitySeverity = 'UNKNOWN_VULNERABILITY_SEVERITY',
}

export enum VulnerabilityState {
    Deferred = 'DEFERRED',
    FalsePositive = 'FALSE_POSITIVE',
    Observed = 'OBSERVED',
}

export type _Service = {
    __typename?: '_Service';
    sdl: Scalars['String'];
};

export type ClustersCountQueryVariables = Exact<{ [key: string]: never }>;

export type ClustersCountQuery = { __typename?: 'Query'; results: number };

export type NodesCountQueryVariables = Exact<{ [key: string]: never }>;

export type NodesCountQuery = { __typename?: 'Query'; results: number };

export type NamespacesCountQueryVariables = Exact<{ [key: string]: never }>;

export type NamespacesCountQuery = { __typename?: 'Query'; results: number };

export type DeploymentsCountQueryVariables = Exact<{ [key: string]: never }>;

export type DeploymentsCountQuery = { __typename?: 'Query'; results: number };

export type GetAllNamespacesByClusterQueryVariables = Exact<{
    query?: InputMaybe<Scalars['String']>;
}>;

export type GetAllNamespacesByClusterQuery = {
    __typename?: 'Query';
    clusters: Array<{
        __typename?: 'Cluster';
        id: string;
        name: string;
        namespaces: Array<{
            __typename?: 'Namespace';
            metadata?: { __typename?: 'NamespaceMetadata'; id: string; name: string } | null;
        }>;
    }>;
};

export type Summary_CountsQueryVariables = Exact<{ [key: string]: never }>;

export type Summary_CountsQuery = {
    __typename?: 'Query';
    clusterCount: number;
    nodeCount: number;
    violationCount: number;
    deploymentCount: number;
    imageCount: number;
    secretCount: number;
};

export type AgingImagesQueryQueryVariables = Exact<{
    query0?: InputMaybe<Scalars['String']>;
    query1?: InputMaybe<Scalars['String']>;
    query2?: InputMaybe<Scalars['String']>;
    query3?: InputMaybe<Scalars['String']>;
}>;

export type AgingImagesQueryQuery = {
    __typename?: 'Query';
    timeRange0: number;
    timeRange1: number;
    timeRange2: number;
    timeRange3: number;
};

export type GetImagesAtMostRiskQueryVariables = Exact<{
    query?: InputMaybe<Scalars['String']>;
}>;

export type GetImagesAtMostRiskQuery = {
    __typename?: 'Query';
    images: Array<{
        __typename?: 'Image';
        id: string;
        priority: number;
        name?: { __typename?: 'ImageName'; remote: string; fullName: string } | null;
        imageVulnerabilityCounter: {
            __typename?: 'VulnerabilityCounter';
            important: {
                __typename?: 'VulnerabilityFixableCounterResolver';
                total: number;
                fixable: number;
            };
            critical: {
                __typename?: 'VulnerabilityFixableCounterResolver';
                total: number;
                fixable: number;
            };
        };
    }>;
};

export type HealthsQueryVariables = Exact<{
    query?: InputMaybe<Scalars['String']>;
}>;

export type HealthsQuery = {
    __typename?: 'Query';
    results: {
        __typename?: 'ClusterHealthCounter';
        total: number;
        uninitialized: number;
        healthy: number;
        degraded: number;
        unhealthy: number;
    };
};

export type GetMitreAttackVectorsQueryVariables = Exact<{
    id: Scalars['ID'];
}>;

export type GetMitreAttackVectorsQuery = {
    __typename?: 'Query';
    policy?: {
        __typename?: 'Policy';
        mitreAttackVectors: Array<{
            __typename?: 'MitreAttackVector';
            tactic?: {
                __typename?: 'MitreTactic';
                id: string;
                name: string;
                description: string;
            } | null;
            techniques: Array<{
                __typename?: 'MitreTechnique';
                id: string;
                name: string;
                description: string;
            } | null>;
        }>;
    } | null;
};

export type GetDeploymentsForPolicyGenerationQueryVariables = Exact<{
    query: Scalars['String'];
    pagination: Pagination;
}>;

export type GetDeploymentsForPolicyGenerationQuery = {
    __typename?: 'Query';
    deployments: Array<{ __typename?: 'Deployment'; id: string; name: string; namespace: string }>;
};

export type GetImageVulnerabilitiesQueryVariables = Exact<{
    imageId: Scalars['ID'];
    vulnsQuery?: InputMaybe<Scalars['String']>;
    pagination?: InputMaybe<Pagination>;
}>;

export type GetImageVulnerabilitiesQuery = {
    __typename?: 'Query';
    image?: {
        __typename?: 'Image';
        vulnCount: number;
        name?: { __typename?: 'ImageName'; registry: string; remote: string; tag: string } | null;
        vulns: Array<{
            __typename?: 'ImageVulnerability';
            id: string;
            cve: string;
            isFixable: boolean;
            severity: string;
            scoreVersion: string;
            cvss: number;
            discoveredAtImage?: string | null;
            components: Array<{
                __typename?: 'ImageComponent';
                id: string;
                name: string;
                version: string;
                fixedIn: string;
            }>;
            vulnerabilityRequest?: {
                __typename?: 'VulnerabilityRequest';
                id: string;
                targetState: string;
                status: string;
                expired: boolean;
                requestor?: { __typename?: 'SlimUser'; id: string; name: string } | null;
                approvers: Array<{ __typename?: 'SlimUser'; id: string; name: string }>;
                comments: Array<{
                    __typename?: 'RequestComment';
                    createdAt?: string | null;
                    id: string;
                    message: string;
                    user?: { __typename?: 'SlimUser'; id: string; name: string } | null;
                }>;
                deferralReq?: {
                    __typename?: 'DeferralRequest';
                    expiresOn?: string | null;
                    expiresWhenFixed: boolean;
                } | null;
                updatedDeferralReq?: {
                    __typename?: 'DeferralRequest';
                    expiresOn?: string | null;
                    expiresWhenFixed: boolean;
                } | null;
                scope?: {
                    __typename?: 'VulnerabilityRequest_Scope';
                    imageScope?: {
                        __typename?: 'VulnerabilityRequest_Scope_Image';
                        registry: string;
                        remote: string;
                        tag: string;
                    } | null;
                } | null;
                cves?: { __typename?: 'VulnerabilityRequest_CVEs'; cves: Array<string> } | null;
            } | null;
        } | null>;
    } | null;
};

export type DeferVulnerabilityMutationVariables = Exact<{
    request: DeferVulnRequest;
}>;

export type DeferVulnerabilityMutation = {
    __typename?: 'Mutation';
    deferVulnerability: { __typename?: 'VulnerabilityRequest'; id: string };
};

export type MarkVulnerabilityFalsePositiveMutationVariables = Exact<{
    request: FalsePositiveVulnRequest;
}>;

export type MarkVulnerabilityFalsePositiveMutation = {
    __typename?: 'Mutation';
    markVulnerabilityFalsePositive: { __typename?: 'VulnerabilityRequest'; id: string };
};

export type GetVulnerabilityRequestsQueryVariables = Exact<{
    query?: InputMaybe<Scalars['String']>;
    requestIDSelector?: InputMaybe<Scalars['String']>;
    pagination?: InputMaybe<Pagination>;
}>;

export type GetVulnerabilityRequestsQuery = {
    __typename?: 'Query';
    vulnerabilityRequestsCount: number;
    vulnerabilityRequests: Array<{
        __typename?: 'VulnerabilityRequest';
        id: string;
        targetState: string;
        status: string;
        deploymentCount: number;
        imageCount: number;
        requestor?: { __typename?: 'SlimUser'; id: string; name: string } | null;
        comments: Array<{
            __typename?: 'RequestComment';
            createdAt?: string | null;
            id: string;
            message: string;
            user?: { __typename?: 'SlimUser'; id: string; name: string } | null;
        }>;
        scope?: {
            __typename?: 'VulnerabilityRequest_Scope';
            imageScope?: {
                __typename?: 'VulnerabilityRequest_Scope_Image';
                registry: string;
                remote: string;
                tag: string;
            } | null;
        } | null;
        deferralReq?: {
            __typename?: 'DeferralRequest';
            expiresOn?: string | null;
            expiresWhenFixed: boolean;
        } | null;
        updatedDeferralReq?: {
            __typename?: 'DeferralRequest';
            expiresOn?: string | null;
            expiresWhenFixed: boolean;
        } | null;
        cves?: { __typename?: 'VulnerabilityRequest_CVEs'; cves: Array<string> } | null;
        deployments: Array<{
            __typename?: 'Deployment';
            id: string;
            name: string;
            namespace: string;
            clusterName: string;
        }>;
        images: Array<{
            __typename?: 'Image';
            id: string;
            name?: { __typename?: 'ImageName'; fullName: string } | null;
        }>;
    }>;
};

export type ApproveVulnerabilityRequestMutationVariables = Exact<{
    requestID: Scalars['ID'];
    comment: Scalars['String'];
}>;

export type ApproveVulnerabilityRequestMutation = {
    __typename?: 'Mutation';
    approveVulnerabilityRequest: { __typename?: 'VulnerabilityRequest'; id: string };
};

export type DenyVulnerabilityRequestMutationVariables = Exact<{
    requestID: Scalars['ID'];
    comment: Scalars['String'];
}>;

export type DenyVulnerabilityRequestMutation = {
    __typename?: 'Mutation';
    denyVulnerabilityRequest: { __typename?: 'VulnerabilityRequest'; id: string };
};

export type DeleteVulnerabilityRequestMutationVariables = Exact<{
    requestID: Scalars['ID'];
}>;

export type DeleteVulnerabilityRequestMutation = {
    __typename?: 'Mutation';
    deleteVulnerabilityRequest: boolean;
};

export type UndoVulnerabilityRequestMutationVariables = Exact<{
    requestID: Scalars['ID'];
}>;

export type UndoVulnerabilityRequestMutation = {
    __typename?: 'Mutation';
    undoVulnerabilityRequest: { __typename?: 'VulnerabilityRequest'; id: string };
};

export type UpdateVulnerabilityRequestMutationVariables = Exact<{
    requestID: Scalars['ID'];
    comment: Scalars['String'];
    expiry: VulnReqExpiry;
}>;

export type UpdateVulnerabilityRequestMutation = {
    __typename?: 'Mutation';
    updateVulnerabilityRequest: { __typename?: 'VulnerabilityRequest'; id: string };
};

export type GetDeploymentMetadataQueryVariables = Exact<{
    id: Scalars['ID'];
}>;

export type GetDeploymentMetadataQuery = {
    __typename?: 'Query';
    deployment?: {
        __typename?: 'Deployment';
        id: string;
        name: string;
        namespace: string;
        clusterName: string;
        created?: string | null;
        imageCount: number;
    } | null;
};

export type DeploymentMetadataFragment = {
    __typename?: 'Deployment';
    id: string;
    name: string;
    namespace: string;
    clusterName: string;
    created?: string | null;
    imageCount: number;
};

export type GetDeploymentResourcesQueryVariables = Exact<{
    id: Scalars['ID'];
    query?: InputMaybe<Scalars['String']>;
    pagination?: InputMaybe<Pagination>;
}>;

export type GetDeploymentResourcesQuery = {
    __typename?: 'Query';
    deployment?: {
        __typename?: 'Deployment';
        id: string;
        imageCount: number;
        images: Array<{
            __typename?: 'Image';
            id: string;
            deploymentCount: number;
            operatingSystem: string;
            scanTime?: string | null;
            name?: {
                __typename?: 'ImageName';
                registry: string;
                remote: string;
                tag: string;
            } | null;
        }>;
    } | null;
};

export type GetDeploymentSummaryDataQueryVariables = Exact<{
    id: Scalars['ID'];
    query: Scalars['String'];
}>;

export type GetDeploymentSummaryDataQuery = {
    __typename?: 'Query';
    deployment?: {
        __typename?: 'Deployment';
        id: string;
        imageCVECountBySeverity: {
            __typename?: 'ResourceCountByCVESeverity';
            low: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
            moderate: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
            important: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
            critical: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
        };
    } | null;
};

export type GetCvesForDeploymentQueryVariables = Exact<{
    id: Scalars['ID'];
    query: Scalars['String'];
    pagination: Pagination;
}>;

export type GetCvesForDeploymentQuery = {
    __typename?: 'Query';
    deployment?: {
        __typename?: 'Deployment';
        imageVulnerabilityCount: number;
        id: string;
        images: Array<{
            __typename?: 'Image';
            id: string;
            name?: {
                __typename?: 'ImageName';
                registry: string;
                remote: string;
                tag: string;
            } | null;
            metadata?: {
                __typename?: 'ImageMetadata';
                v1?: {
                    __typename?: 'V1Metadata';
                    layers: Array<{
                        __typename?: 'ImageLayer';
                        instruction: string;
                        value: string;
                    } | null>;
                } | null;
            } | null;
        }>;
        imageVulnerabilities: Array<{
            __typename?: 'ImageVulnerability';
            cve: string;
            summary: string;
            vulnerabilityId: string;
            images: Array<{
                __typename?: 'Image';
                imageId: string;
                imageComponents: Array<{
                    __typename?: 'ImageComponent';
                    name: string;
                    version: string;
                    location: string;
                    source: SourceType;
                    layerIndex?: number | null;
                    imageVulnerabilities: Array<{
                        __typename?: 'ImageVulnerability';
                        severity: string;
                        cvss: number;
                        scoreVersion: string;
                        fixedByVersion: string;
                        discoveredAtImage?: string | null;
                        vulnerabilityId: string;
                    } | null>;
                }>;
            }>;
        }>;
    } | null;
};

export type ImageResourcesFragment = {
    __typename?: 'Deployment';
    imageCount: number;
    images: Array<{
        __typename?: 'Image';
        id: string;
        deploymentCount: number;
        operatingSystem: string;
        scanTime?: string | null;
        name?: { __typename?: 'ImageName'; registry: string; remote: string; tag: string } | null;
    }>;
};

export type DeploymentResourcesFragment = {
    __typename?: 'Image';
    deploymentCount: number;
    deployments: Array<{
        __typename?: 'Deployment';
        id: string;
        name: string;
        clusterName: string;
        namespace: string;
        created?: string | null;
    }>;
};

export type GetImageDetailsQueryVariables = Exact<{
    id: Scalars['ID'];
}>;

export type GetImageDetailsQuery = {
    __typename?: 'Query';
    image?: {
        __typename?: 'Image';
        id: string;
        deploymentCount: number;
        operatingSystem: string;
        scanTime?: string | null;
        name?: { __typename?: 'ImageName'; registry: string; remote: string; tag: string } | null;
        metadata?: {
            __typename?: 'ImageMetadata';
            v1?: { __typename?: 'V1Metadata'; created?: string | null } | null;
        } | null;
        dataSource?: { __typename?: 'DataSource'; name: string } | null;
    } | null;
};

export type GetImageResourcesQueryVariables = Exact<{
    id: Scalars['ID'];
    query?: InputMaybe<Scalars['String']>;
    pagination?: InputMaybe<Pagination>;
}>;

export type GetImageResourcesQuery = {
    __typename?: 'Query';
    image?: {
        __typename?: 'Image';
        id: string;
        deploymentCount: number;
        deployments: Array<{
            __typename?: 'Deployment';
            id: string;
            name: string;
            clusterName: string;
            namespace: string;
            created?: string | null;
        }>;
    } | null;
};

export type GetCvEsForImageQueryVariables = Exact<{
    id: Scalars['ID'];
    query: Scalars['String'];
    pagination: Pagination;
}>;

export type GetCvEsForImageQuery = {
    __typename?: 'Query';
    image?: {
        __typename?: 'Image';
        id: string;
        imageCVECountBySeverity: {
            __typename?: 'ResourceCountByCVESeverity';
            low: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
            moderate: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
            important: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
            critical: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
        };
        imageVulnerabilities: Array<{
            __typename?: 'ImageVulnerability';
            severity: string;
            cve: string;
            summary: string;
            cvss: number;
            scoreVersion: string;
            discoveredAtImage?: string | null;
            imageComponents: Array<{
                __typename?: 'ImageComponent';
                name: string;
                version: string;
                location: string;
                source: SourceType;
                layerIndex?: number | null;
                imageVulnerabilities: Array<{
                    __typename?: 'ImageVulnerability';
                    severity: string;
                    fixedByVersion: string;
                    vulnerabilityId: string;
                } | null>;
            }>;
        } | null>;
        name?: { __typename?: 'ImageName'; registry: string; remote: string; tag: string } | null;
        metadata?: {
            __typename?: 'ImageMetadata';
            v1?: {
                __typename?: 'V1Metadata';
                layers: Array<{
                    __typename?: 'ImageLayer';
                    instruction: string;
                    value: string;
                } | null>;
            } | null;
        } | null;
    } | null;
};

export type GetImageCveMetadataQueryVariables = Exact<{
    cve: Scalars['String'];
}>;

export type GetImageCveMetadataQuery = {
    __typename?: 'Query';
    imageCVE?: {
        __typename?: 'ImageCVECore';
        cve: string;
        firstDiscoveredInSystem?: string | null;
        distroTuples: Array<{
            __typename?: 'ImageVulnerability';
            summary: string;
            link: string;
            operatingSystem: string;
        }>;
    } | null;
};

export type GetImageCveSummaryDataQueryVariables = Exact<{
    cve: Scalars['String'];
    query: Scalars['String'];
}>;

export type GetImageCveSummaryDataQuery = {
    __typename?: 'Query';
    imageCount: number;
    deploymentCount: number;
    totalImageCount: number;
    imageCVE?: {
        __typename?: 'ImageCVECore';
        cve: string;
        affectedImageCount: number;
        affectedImageCountBySeverity: {
            __typename?: 'ResourceCountByCVESeverity';
            low: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
            moderate: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
            important: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
            critical: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
        };
    } | null;
};

export type GetImagesForCveQueryVariables = Exact<{
    query?: InputMaybe<Scalars['String']>;
    pagination?: InputMaybe<Pagination>;
}>;

export type GetImagesForCveQuery = {
    __typename?: 'Query';
    images: Array<{
        __typename?: 'Image';
        operatingSystem: string;
        watchStatus: ImageWatchStatus;
        scanTime?: string | null;
        id: string;
        imageComponents: Array<{
            __typename?: 'ImageComponent';
            name: string;
            version: string;
            location: string;
            source: SourceType;
            layerIndex?: number | null;
            imageVulnerabilities: Array<{
                __typename?: 'ImageVulnerability';
                cvss: number;
                scoreVersion: string;
                severity: string;
                fixedByVersion: string;
                vulnerabilityId: string;
            } | null>;
        }>;
        name?: { __typename?: 'ImageName'; registry: string; remote: string; tag: string } | null;
        metadata?: {
            __typename?: 'ImageMetadata';
            v1?: {
                __typename?: 'V1Metadata';
                layers: Array<{
                    __typename?: 'ImageLayer';
                    instruction: string;
                    value: string;
                } | null>;
            } | null;
        } | null;
    }>;
};

export type GetDeploymentsForCveQueryVariables = Exact<{
    query?: InputMaybe<Scalars['String']>;
    pagination?: InputMaybe<Pagination>;
    lowImageCountQuery?: InputMaybe<Scalars['String']>;
    moderateImageCountQuery?: InputMaybe<Scalars['String']>;
    importantImageCountQuery?: InputMaybe<Scalars['String']>;
    criticalImageCountQuery?: InputMaybe<Scalars['String']>;
}>;

export type GetDeploymentsForCveQuery = {
    __typename?: 'Query';
    deployments: Array<{
        __typename?: 'Deployment';
        id: string;
        name: string;
        namespace: string;
        clusterName: string;
        created?: string | null;
        lowImageCount: number;
        moderateImageCount: number;
        importantImageCount: number;
        criticalImageCount: number;
        images: Array<{
            __typename?: 'Image';
            id: string;
            imageComponents: Array<{
                __typename?: 'ImageComponent';
                name: string;
                version: string;
                location: string;
                source: SourceType;
                layerIndex?: number | null;
                imageVulnerabilities: Array<{
                    __typename?: 'ImageVulnerability';
                    severity: string;
                    cvss: number;
                    scoreVersion: string;
                    fixedByVersion: string;
                    discoveredAtImage?: string | null;
                    vulnerabilityId: string;
                } | null>;
            }>;
            name?: {
                __typename?: 'ImageName';
                registry: string;
                remote: string;
                tag: string;
            } | null;
            metadata?: {
                __typename?: 'ImageMetadata';
                v1?: {
                    __typename?: 'V1Metadata';
                    layers: Array<{
                        __typename?: 'ImageLayer';
                        instruction: string;
                        value: string;
                    } | null>;
                } | null;
            } | null;
        }>;
    }>;
};

export type ImageCveMetadataFragment = {
    __typename?: 'ImageCVECore';
    cve: string;
    firstDiscoveredInSystem?: string | null;
    distroTuples: Array<{
        __typename?: 'ImageVulnerability';
        summary: string;
        link: string;
        operatingSystem: string;
    }>;
};

export type ResourceCountsByCveSeverityAndStatusFragment = {
    __typename?: 'ResourceCountByCVESeverity';
    low: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
    moderate: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
    important: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
    critical: { __typename?: 'ResourceCountByFixability'; total: number; fixable: number };
};

export type DeploymentsForCveFragment = {
    __typename?: 'Deployment';
    id: string;
    name: string;
    namespace: string;
    clusterName: string;
    created?: string | null;
    lowImageCount: number;
    moderateImageCount: number;
    importantImageCount: number;
    criticalImageCount: number;
    images: Array<{
        __typename?: 'Image';
        id: string;
        imageComponents: Array<{
            __typename?: 'ImageComponent';
            name: string;
            version: string;
            location: string;
            source: SourceType;
            layerIndex?: number | null;
            imageVulnerabilities: Array<{
                __typename?: 'ImageVulnerability';
                severity: string;
                cvss: number;
                scoreVersion: string;
                fixedByVersion: string;
                discoveredAtImage?: string | null;
                vulnerabilityId: string;
            } | null>;
        }>;
        name?: { __typename?: 'ImageName'; registry: string; remote: string; tag: string } | null;
        metadata?: {
            __typename?: 'ImageMetadata';
            v1?: {
                __typename?: 'V1Metadata';
                layers: Array<{
                    __typename?: 'ImageLayer';
                    instruction: string;
                    value: string;
                } | null>;
            } | null;
        } | null;
    }>;
};

export type ImagesForCveFragment = {
    __typename?: 'Image';
    operatingSystem: string;
    watchStatus: ImageWatchStatus;
    scanTime?: string | null;
    id: string;
    imageComponents: Array<{
        __typename?: 'ImageComponent';
        name: string;
        version: string;
        location: string;
        source: SourceType;
        layerIndex?: number | null;
        imageVulnerabilities: Array<{
            __typename?: 'ImageVulnerability';
            cvss: number;
            scoreVersion: string;
            severity: string;
            fixedByVersion: string;
            vulnerabilityId: string;
        } | null>;
    }>;
    name?: { __typename?: 'ImageName'; registry: string; remote: string; tag: string } | null;
    metadata?: {
        __typename?: 'ImageMetadata';
        v1?: {
            __typename?: 'V1Metadata';
            layers: Array<{ __typename?: 'ImageLayer'; instruction: string; value: string } | null>;
        } | null;
    } | null;
};

export type GetImageCveListQueryVariables = Exact<{
    query?: InputMaybe<Scalars['String']>;
    pagination?: InputMaybe<Pagination>;
}>;

export type GetImageCveListQuery = {
    __typename?: 'Query';
    imageCVEs: Array<{
        __typename?: 'ImageCVECore';
        cve: string;
        topCVSS: number;
        affectedImageCount: number;
        firstDiscoveredInSystem?: string | null;
        affectedImageCountBySeverity: {
            __typename?: 'ResourceCountByCVESeverity';
            critical: { __typename?: 'ResourceCountByFixability'; total: number };
            important: { __typename?: 'ResourceCountByFixability'; total: number };
            moderate: { __typename?: 'ResourceCountByFixability'; total: number };
            low: { __typename?: 'ResourceCountByFixability'; total: number };
        };
        distroTuples: Array<{
            __typename?: 'ImageVulnerability';
            summary: string;
            operatingSystem: string;
            cvss: number;
            scoreVersion: string;
        }>;
    }>;
};

export type GetUnfilteredImageCountQueryVariables = Exact<{ [key: string]: never }>;

export type GetUnfilteredImageCountQuery = { __typename?: 'Query'; imageCount: number };

export type DeploymentComponentVulnerabilitiesFragment = {
    __typename?: 'ImageComponent';
    name: string;
    version: string;
    location: string;
    source: SourceType;
    layerIndex?: number | null;
    imageVulnerabilities: Array<{
        __typename?: 'ImageVulnerability';
        severity: string;
        cvss: number;
        scoreVersion: string;
        fixedByVersion: string;
        discoveredAtImage?: string | null;
        vulnerabilityId: string;
    } | null>;
};

export type DeploymentWithVulnerabilitiesFragment = {
    __typename?: 'Deployment';
    id: string;
    images: Array<{
        __typename?: 'Image';
        id: string;
        name?: { __typename?: 'ImageName'; registry: string; remote: string; tag: string } | null;
        metadata?: {
            __typename?: 'ImageMetadata';
            v1?: {
                __typename?: 'V1Metadata';
                layers: Array<{
                    __typename?: 'ImageLayer';
                    instruction: string;
                    value: string;
                } | null>;
            } | null;
        } | null;
    }>;
    imageVulnerabilities: Array<{
        __typename?: 'ImageVulnerability';
        cve: string;
        summary: string;
        vulnerabilityId: string;
        images: Array<{
            __typename?: 'Image';
            imageId: string;
            imageComponents: Array<{
                __typename?: 'ImageComponent';
                name: string;
                version: string;
                location: string;
                source: SourceType;
                layerIndex?: number | null;
                imageVulnerabilities: Array<{
                    __typename?: 'ImageVulnerability';
                    severity: string;
                    cvss: number;
                    scoreVersion: string;
                    fixedByVersion: string;
                    discoveredAtImage?: string | null;
                    vulnerabilityId: string;
                } | null>;
            }>;
        }>;
    }>;
};

export type GetDeploymentListQueryVariables = Exact<{
    query?: InputMaybe<Scalars['String']>;
    pagination?: InputMaybe<Pagination>;
}>;

export type GetDeploymentListQuery = {
    __typename?: 'Query';
    deployments: Array<{
        __typename?: 'Deployment';
        id: string;
        name: string;
        clusterName: string;
        namespace: string;
        imageCount: number;
        created?: string | null;
        imageCVECountBySeverity: {
            __typename?: 'ResourceCountByCVESeverity';
            critical: { __typename?: 'ResourceCountByFixability'; total: number };
            important: { __typename?: 'ResourceCountByFixability'; total: number };
            moderate: { __typename?: 'ResourceCountByFixability'; total: number };
            low: { __typename?: 'ResourceCountByFixability'; total: number };
        };
    }>;
};

export type ImageComponentVulnerabilitiesFragment = {
    __typename?: 'ImageComponent';
    name: string;
    version: string;
    location: string;
    source: SourceType;
    layerIndex?: number | null;
    imageVulnerabilities: Array<{
        __typename?: 'ImageVulnerability';
        severity: string;
        fixedByVersion: string;
        vulnerabilityId: string;
    } | null>;
};

export type ImageVulnerabilityFieldsFragment = {
    __typename?: 'ImageVulnerability';
    severity: string;
    cve: string;
    summary: string;
    cvss: number;
    scoreVersion: string;
    discoveredAtImage?: string | null;
    imageComponents: Array<{
        __typename?: 'ImageComponent';
        name: string;
        version: string;
        location: string;
        source: SourceType;
        layerIndex?: number | null;
        imageVulnerabilities: Array<{
            __typename?: 'ImageVulnerability';
            severity: string;
            fixedByVersion: string;
            vulnerabilityId: string;
        } | null>;
    }>;
};

export type GetImageListQueryVariables = Exact<{
    query?: InputMaybe<Scalars['String']>;
    pagination?: InputMaybe<Pagination>;
}>;

export type GetImageListQuery = {
    __typename?: 'Query';
    images: Array<{
        __typename?: 'Image';
        id: string;
        operatingSystem: string;
        deploymentCount: number;
        watchStatus: ImageWatchStatus;
        scanTime?: string | null;
        name?: { __typename?: 'ImageName'; registry: string; remote: string; tag: string } | null;
        imageCVECountBySeverity: {
            __typename?: 'ResourceCountByCVESeverity';
            critical: { __typename?: 'ResourceCountByFixability'; total: number };
            important: { __typename?: 'ResourceCountByFixability'; total: number };
            moderate: { __typename?: 'ResourceCountByFixability'; total: number };
            low: { __typename?: 'ResourceCountByFixability'; total: number };
        };
        metadata?: {
            __typename?: 'ImageMetadata';
            v1?: { __typename?: 'V1Metadata'; created?: string | null } | null;
        } | null;
    }>;
};

export type ImageMetadataContextFragment = {
    __typename?: 'Image';
    id: string;
    name?: { __typename?: 'ImageName'; registry: string; remote: string; tag: string } | null;
    metadata?: {
        __typename?: 'ImageMetadata';
        v1?: {
            __typename?: 'V1Metadata';
            layers: Array<{ __typename?: 'ImageLayer'; instruction: string; value: string } | null>;
        } | null;
    } | null;
};

export type GetEntityTypeCountsQueryVariables = Exact<{
    query?: InputMaybe<Scalars['String']>;
}>;

export type GetEntityTypeCountsQuery = {
    __typename?: 'Query';
    imageCount: number;
    deploymentCount: number;
    imageCVECount: number;
};

export type ImageDetailsFragment = {
    __typename?: 'Image';
    deploymentCount: number;
    operatingSystem: string;
    scanTime?: string | null;
    metadata?: {
        __typename?: 'ImageMetadata';
        v1?: { __typename?: 'V1Metadata'; created?: string | null } | null;
    } | null;
    dataSource?: { __typename?: 'DataSource'; name: string } | null;
};

export type GetDeploymentCountQueryVariables = Exact<{
    query?: InputMaybe<Scalars['String']>;
}>;

export type GetDeploymentCountQuery = { __typename?: 'Query'; count: number };

export const DeploymentMetadataFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentMetadata' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Deployment' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'namespace' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'clusterName' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'created' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'imageCount' } },
                ],
            },
        },
    ],
} as unknown as DocumentNode<DeploymentMetadataFragment, unknown>;
export const ImageResourcesFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageResources' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Deployment' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'images' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'name' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'registry' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'remote' },
                                            },
                                            { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'deploymentCount' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'operatingSystem' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'scanTime' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<ImageResourcesFragment, unknown>;
export const DeploymentResourcesFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentResources' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deploymentCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deployments' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'clusterName' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'namespace' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'created' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<DeploymentResourcesFragment, unknown>;
export const ImageCveMetadataFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageCVEMetadata' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'ImageCVECore' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'cve' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'firstDiscoveredInSystem' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'distroTuples' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'summary' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'link' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'operatingSystem' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<ImageCveMetadataFragment, unknown>;
export const ResourceCountsByCveSeverityAndStatusFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ResourceCountsByCVESeverityAndStatus' },
            typeCondition: {
                kind: 'NamedType',
                name: { kind: 'Name', value: 'ResourceCountByCVESeverity' },
            },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'low' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'moderate' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'important' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'critical' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<ResourceCountsByCveSeverityAndStatusFragment, unknown>;
export const ImageMetadataContextFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageMetadataContext' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'name' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'registry' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'remote' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'metadata' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'v1' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'layers' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'instruction',
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'value' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<ImageMetadataContextFragment, unknown>;
export const DeploymentComponentVulnerabilitiesFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentComponentVulnerabilities' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'ImageComponent' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'version' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'location' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'source' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'layerIndex' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageVulnerabilities' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnerabilityId' },
                                    name: { kind: 'Name', value: 'id' },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'severity' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'cvss' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'scoreVersion' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixedByVersion' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'discoveredAtImage' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<DeploymentComponentVulnerabilitiesFragment, unknown>;
export const DeploymentsForCveFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentsForCVE' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Deployment' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'namespace' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'clusterName' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'created' } },
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'lowImageCount' },
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'lowImageCountQuery' },
                                },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'moderateImageCount' },
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'moderateImageCountQuery' },
                                },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'importantImageCount' },
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'importantImageCountQuery' },
                                },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'criticalImageCount' },
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'criticalImageCountQuery' },
                                },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'images' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImageMetadataContext' },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageComponents' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'FragmentSpread',
                                                name: {
                                                    kind: 'Name',
                                                    value: 'DeploymentComponentVulnerabilities',
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageMetadataContext' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'name' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'registry' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'remote' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'metadata' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'v1' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'layers' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'instruction',
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'value' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentComponentVulnerabilities' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'ImageComponent' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'version' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'location' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'source' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'layerIndex' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageVulnerabilities' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnerabilityId' },
                                    name: { kind: 'Name', value: 'id' },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'severity' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'cvss' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'scoreVersion' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixedByVersion' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'discoveredAtImage' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<DeploymentsForCveFragment, unknown>;
export const ImageComponentVulnerabilitiesFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageComponentVulnerabilities' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'ImageComponent' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'version' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'location' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'source' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'layerIndex' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageVulnerabilities' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnerabilityId' },
                                    name: { kind: 'Name', value: 'id' },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'severity' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixedByVersion' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<ImageComponentVulnerabilitiesFragment, unknown>;
export const ImagesForCveFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImagesForCVE' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'FragmentSpread',
                        name: { kind: 'Name', value: 'ImageMetadataContext' },
                    },
                    { kind: 'Field', name: { kind: 'Name', value: 'operatingSystem' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'watchStatus' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'scanTime' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageComponents' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageVulnerabilities' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'cvss' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'scoreVersion' },
                                            },
                                        ],
                                    },
                                },
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImageComponentVulnerabilities' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageMetadataContext' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'name' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'registry' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'remote' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'metadata' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'v1' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'layers' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'instruction',
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'value' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageComponentVulnerabilities' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'ImageComponent' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'version' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'location' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'source' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'layerIndex' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageVulnerabilities' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnerabilityId' },
                                    name: { kind: 'Name', value: 'id' },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'severity' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixedByVersion' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<ImagesForCveFragment, unknown>;
export const DeploymentWithVulnerabilitiesFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentWithVulnerabilities' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Deployment' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'images' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImageMetadataContext' },
                                },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageVulnerabilities' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnerabilityId' },
                                    name: { kind: 'Name', value: 'id' },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'cve' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'summary' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'images' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                alias: { kind: 'Name', value: 'imageId' },
                                                name: { kind: 'Name', value: 'id' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'imageComponents' },
                                                arguments: [
                                                    {
                                                        kind: 'Argument',
                                                        name: { kind: 'Name', value: 'query' },
                                                        value: {
                                                            kind: 'Variable',
                                                            name: { kind: 'Name', value: 'query' },
                                                        },
                                                    },
                                                ],
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'FragmentSpread',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'DeploymentComponentVulnerabilities',
                                                            },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageMetadataContext' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'name' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'registry' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'remote' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'metadata' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'v1' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'layers' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'instruction',
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'value' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentComponentVulnerabilities' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'ImageComponent' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'version' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'location' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'source' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'layerIndex' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageVulnerabilities' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnerabilityId' },
                                    name: { kind: 'Name', value: 'id' },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'severity' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'cvss' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'scoreVersion' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixedByVersion' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'discoveredAtImage' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<DeploymentWithVulnerabilitiesFragment, unknown>;
export const ImageVulnerabilityFieldsFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageVulnerabilityFields' },
            typeCondition: {
                kind: 'NamedType',
                name: { kind: 'Name', value: 'ImageVulnerability' },
            },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'severity' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'cve' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'summary' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'cvss' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'scoreVersion' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'discoveredAtImage' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageComponents' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImageComponentVulnerabilities' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageComponentVulnerabilities' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'ImageComponent' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'version' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'location' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'source' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'layerIndex' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageVulnerabilities' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnerabilityId' },
                                    name: { kind: 'Name', value: 'id' },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'severity' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixedByVersion' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<ImageVulnerabilityFieldsFragment, unknown>;
export const ImageDetailsFragmentDoc = {
    kind: 'Document',
    definitions: [
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageDetails' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'deploymentCount' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'operatingSystem' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'metadata' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'v1' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'created' },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'dataSource' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [{ kind: 'Field', name: { kind: 'Name', value: 'name' } }],
                        },
                    },
                    { kind: 'Field', name: { kind: 'Name', value: 'scanTime' } },
                ],
            },
        },
    ],
} as unknown as DocumentNode<ImageDetailsFragment, unknown>;
export const ClustersCountDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'clustersCount' },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'results' },
                        name: { kind: 'Name', value: 'complianceClusterCount' },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<ClustersCountQuery, ClustersCountQueryVariables>;
export const NodesCountDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'nodesCount' },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'results' },
                        name: { kind: 'Name', value: 'complianceNodeCount' },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<NodesCountQuery, NodesCountQueryVariables>;
export const NamespacesCountDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'namespacesCount' },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'results' },
                        name: { kind: 'Name', value: 'complianceNamespaceCount' },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<NamespacesCountQuery, NamespacesCountQueryVariables>;
export const DeploymentsCountDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'deploymentsCount' },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'results' },
                        name: { kind: 'Name', value: 'complianceDeploymentCount' },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<DeploymentsCountQuery, DeploymentsCountQueryVariables>;
export const GetAllNamespacesByClusterDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getAllNamespacesByCluster' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'clusters' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'namespaces' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'metadata' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'id' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'name' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<
    GetAllNamespacesByClusterQuery,
    GetAllNamespacesByClusterQueryVariables
>;
export const Summary_CountsDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'summary_counts' },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'clusterCount' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'nodeCount' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'violationCount' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'deploymentCount' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'imageCount' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'secretCount' } },
                ],
            },
        },
    ],
} as unknown as DocumentNode<Summary_CountsQuery, Summary_CountsQueryVariables>;
export const AgingImagesQueryDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'agingImagesQuery' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query0' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query1' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query2' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query3' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'timeRange0' },
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'query0' },
                                },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'timeRange1' },
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'query1' },
                                },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'timeRange2' },
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'query2' },
                                },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'timeRange3' },
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'query3' },
                                },
                            },
                        ],
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<AgingImagesQueryQuery, AgingImagesQueryQueryVariables>;
export const GetImagesAtMostRiskDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getImagesAtMostRisk' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'images' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'ObjectValue',
                                    fields: [
                                        {
                                            kind: 'ObjectField',
                                            name: { kind: 'Name', value: 'limit' },
                                            value: { kind: 'IntValue', value: '6' },
                                        },
                                        {
                                            kind: 'ObjectField',
                                            name: { kind: 'Name', value: 'sortOption' },
                                            value: {
                                                kind: 'ObjectValue',
                                                fields: [
                                                    {
                                                        kind: 'ObjectField',
                                                        name: { kind: 'Name', value: 'field' },
                                                        value: {
                                                            kind: 'StringValue',
                                                            value: 'Image Risk Priority',
                                                            block: false,
                                                        },
                                                    },
                                                    {
                                                        kind: 'ObjectField',
                                                        name: { kind: 'Name', value: 'reversed' },
                                                        value: {
                                                            kind: 'BooleanValue',
                                                            value: false,
                                                        },
                                                    },
                                                ],
                                            },
                                        },
                                    ],
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'name' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'remote' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'fullName' },
                                            },
                                        ],
                                    },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'priority' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageVulnerabilityCounter' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'important' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'fixable',
                                                            },
                                                        },
                                                    ],
                                                },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'critical' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'fixable',
                                                            },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetImagesAtMostRiskQuery, GetImagesAtMostRiskQueryVariables>;
export const HealthsDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'healths' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'results' },
                        name: { kind: 'Name', value: 'clusterHealthCounter' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'uninitialized' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'healthy' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'degraded' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'unhealthy' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<HealthsQuery, HealthsQueryVariables>;
export const GetMitreAttackVectorsDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getMitreAttackVectors' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'policy' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'id' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'mitreAttackVectors' },
                                    name: { kind: 'Name', value: 'fullMitreAttackVectors' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'tactic' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'id' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'name' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'description',
                                                            },
                                                        },
                                                    ],
                                                },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'techniques' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'id' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'name' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'description',
                                                            },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetMitreAttackVectorsQuery, GetMitreAttackVectorsQueryVariables>;
export const GetDeploymentsForPolicyGenerationDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getDeploymentsForPolicyGeneration' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'pagination' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'Pagination' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deployments' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'namespace' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<
    GetDeploymentsForPolicyGenerationQuery,
    GetDeploymentsForPolicyGenerationQueryVariables
>;
export const GetImageVulnerabilitiesDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getImageVulnerabilities' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'imageId' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'vulnsQuery' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'pagination' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'Pagination' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'image' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'id' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'imageId' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'name' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'registry' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'remote' },
                                            },
                                            { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnCount' },
                                    name: { kind: 'Name', value: 'imageVulnerabilityCount' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'vulnsQuery' },
                                            },
                                        },
                                    ],
                                },
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulns' },
                                    name: { kind: 'Name', value: 'imageVulnerabilities' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'vulnsQuery' },
                                            },
                                        },
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'pagination' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'pagination' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                            { kind: 'Field', name: { kind: 'Name', value: 'cve' } },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'isFixable' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'severity' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'scoreVersion' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'cvss' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'discoveredAtImage' },
                                            },
                                            {
                                                kind: 'Field',
                                                alias: { kind: 'Name', value: 'components' },
                                                name: { kind: 'Name', value: 'imageComponents' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'id' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'name' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'version',
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'fixedIn',
                                                            },
                                                        },
                                                    ],
                                                },
                                            },
                                            {
                                                kind: 'Field',
                                                alias: {
                                                    kind: 'Name',
                                                    value: 'vulnerabilityRequest',
                                                },
                                                name: {
                                                    kind: 'Name',
                                                    value: 'effectiveVulnerabilityRequest',
                                                },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'id' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'targetState',
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'status' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'expired',
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'requestor',
                                                            },
                                                            selectionSet: {
                                                                kind: 'SelectionSet',
                                                                selections: [
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'id',
                                                                        },
                                                                    },
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'name',
                                                                        },
                                                                    },
                                                                ],
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'approvers',
                                                            },
                                                            selectionSet: {
                                                                kind: 'SelectionSet',
                                                                selections: [
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'id',
                                                                        },
                                                                    },
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'name',
                                                                        },
                                                                    },
                                                                ],
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'comments',
                                                            },
                                                            selectionSet: {
                                                                kind: 'SelectionSet',
                                                                selections: [
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'createdAt',
                                                                        },
                                                                    },
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'id',
                                                                        },
                                                                    },
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'message',
                                                                        },
                                                                    },
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'user',
                                                                        },
                                                                        selectionSet: {
                                                                            kind: 'SelectionSet',
                                                                            selections: [
                                                                                {
                                                                                    kind: 'Field',
                                                                                    name: {
                                                                                        kind: 'Name',
                                                                                        value: 'id',
                                                                                    },
                                                                                },
                                                                                {
                                                                                    kind: 'Field',
                                                                                    name: {
                                                                                        kind: 'Name',
                                                                                        value: 'name',
                                                                                    },
                                                                                },
                                                                            ],
                                                                        },
                                                                    },
                                                                ],
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'deferralReq',
                                                            },
                                                            selectionSet: {
                                                                kind: 'SelectionSet',
                                                                selections: [
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'expiresOn',
                                                                        },
                                                                    },
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'expiresWhenFixed',
                                                                        },
                                                                    },
                                                                ],
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'updatedDeferralReq',
                                                            },
                                                            selectionSet: {
                                                                kind: 'SelectionSet',
                                                                selections: [
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'expiresOn',
                                                                        },
                                                                    },
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'expiresWhenFixed',
                                                                        },
                                                                    },
                                                                ],
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'scope' },
                                                            selectionSet: {
                                                                kind: 'SelectionSet',
                                                                selections: [
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'imageScope',
                                                                        },
                                                                        selectionSet: {
                                                                            kind: 'SelectionSet',
                                                                            selections: [
                                                                                {
                                                                                    kind: 'Field',
                                                                                    name: {
                                                                                        kind: 'Name',
                                                                                        value: 'registry',
                                                                                    },
                                                                                },
                                                                                {
                                                                                    kind: 'Field',
                                                                                    name: {
                                                                                        kind: 'Name',
                                                                                        value: 'remote',
                                                                                    },
                                                                                },
                                                                                {
                                                                                    kind: 'Field',
                                                                                    name: {
                                                                                        kind: 'Name',
                                                                                        value: 'tag',
                                                                                    },
                                                                                },
                                                                            ],
                                                                        },
                                                                    },
                                                                ],
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'cves' },
                                                            selectionSet: {
                                                                kind: 'SelectionSet',
                                                                selections: [
                                                                    {
                                                                        kind: 'Field',
                                                                        name: {
                                                                            kind: 'Name',
                                                                            value: 'cves',
                                                                        },
                                                                    },
                                                                ],
                                                            },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetImageVulnerabilitiesQuery, GetImageVulnerabilitiesQueryVariables>;
export const DeferVulnerabilityDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'mutation',
            name: { kind: 'Name', value: 'deferVulnerability' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'request' } },
                    type: {
                        kind: 'NonNullType',
                        type: {
                            kind: 'NamedType',
                            name: { kind: 'Name', value: 'DeferVulnRequest' },
                        },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deferVulnerability' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'request' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'request' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [{ kind: 'Field', name: { kind: 'Name', value: 'id' } }],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<DeferVulnerabilityMutation, DeferVulnerabilityMutationVariables>;
export const MarkVulnerabilityFalsePositiveDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'mutation',
            name: { kind: 'Name', value: 'markVulnerabilityFalsePositive' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'request' } },
                    type: {
                        kind: 'NonNullType',
                        type: {
                            kind: 'NamedType',
                            name: { kind: 'Name', value: 'FalsePositiveVulnRequest' },
                        },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'markVulnerabilityFalsePositive' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'request' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'request' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [{ kind: 'Field', name: { kind: 'Name', value: 'id' } }],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<
    MarkVulnerabilityFalsePositiveMutation,
    MarkVulnerabilityFalsePositiveMutationVariables
>;
export const GetVulnerabilityRequestsDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getVulnerabilityRequests' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: {
                        kind: 'Variable',
                        name: { kind: 'Name', value: 'requestIDSelector' },
                    },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'pagination' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'Pagination' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'vulnerabilityRequests' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'requestIDSelector' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'requestIDSelector' },
                                },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'targetState' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'status' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'requestor' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'name' },
                                            },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'comments' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'createdAt' },
                                            },
                                            { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'message' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'user' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'id' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'name' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'scope' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'imageScope' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'registry',
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'remote' },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'tag' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'deferralReq' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'expiresOn' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'expiresWhenFixed' },
                                            },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'updatedDeferralReq' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'expiresOn' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'expiresWhenFixed' },
                                            },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'cves' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'cves' },
                                            },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'deployments' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'name' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'namespace' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'clusterName' },
                                            },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'deploymentCount' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'images' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'name' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'fullName',
                                                            },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageCount' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'vulnerabilityRequestsCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetVulnerabilityRequestsQuery, GetVulnerabilityRequestsQueryVariables>;
export const ApproveVulnerabilityRequestDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'mutation',
            name: { kind: 'Name', value: 'approveVulnerabilityRequest' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'requestID' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'comment' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'approveVulnerabilityRequest' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'requestID' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'requestID' },
                                },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'comment' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'comment' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [{ kind: 'Field', name: { kind: 'Name', value: 'id' } }],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<
    ApproveVulnerabilityRequestMutation,
    ApproveVulnerabilityRequestMutationVariables
>;
export const DenyVulnerabilityRequestDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'mutation',
            name: { kind: 'Name', value: 'denyVulnerabilityRequest' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'requestID' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'comment' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'denyVulnerabilityRequest' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'requestID' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'requestID' },
                                },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'comment' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'comment' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [{ kind: 'Field', name: { kind: 'Name', value: 'id' } }],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<
    DenyVulnerabilityRequestMutation,
    DenyVulnerabilityRequestMutationVariables
>;
export const DeleteVulnerabilityRequestDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'mutation',
            name: { kind: 'Name', value: 'deleteVulnerabilityRequest' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'requestID' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deleteVulnerabilityRequest' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'requestID' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'requestID' },
                                },
                            },
                        ],
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<
    DeleteVulnerabilityRequestMutation,
    DeleteVulnerabilityRequestMutationVariables
>;
export const UndoVulnerabilityRequestDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'mutation',
            name: { kind: 'Name', value: 'undoVulnerabilityRequest' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'requestID' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'undoVulnerabilityRequest' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'requestID' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'requestID' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [{ kind: 'Field', name: { kind: 'Name', value: 'id' } }],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<
    UndoVulnerabilityRequestMutation,
    UndoVulnerabilityRequestMutationVariables
>;
export const UpdateVulnerabilityRequestDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'mutation',
            name: { kind: 'Name', value: 'updateVulnerabilityRequest' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'requestID' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'comment' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'expiry' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'VulnReqExpiry' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'updateVulnerabilityRequest' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'requestID' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'requestID' },
                                },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'comment' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'comment' },
                                },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'expiry' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'expiry' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [{ kind: 'Field', name: { kind: 'Name', value: 'id' } }],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<
    UpdateVulnerabilityRequestMutation,
    UpdateVulnerabilityRequestMutationVariables
>;
export const GetDeploymentMetadataDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getDeploymentMetadata' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deployment' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'id' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'DeploymentMetadata' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentMetadata' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Deployment' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'namespace' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'clusterName' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'created' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'imageCount' } },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetDeploymentMetadataQuery, GetDeploymentMetadataQueryVariables>;
export const GetDeploymentResourcesDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getDeploymentResources' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'pagination' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'Pagination' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deployment' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'id' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImageResources' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageResources' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Deployment' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'images' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'name' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'registry' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'remote' },
                                            },
                                            { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'deploymentCount' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'operatingSystem' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'scanTime' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetDeploymentResourcesQuery, GetDeploymentResourcesQueryVariables>;
export const GetDeploymentSummaryDataDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getDeploymentSummaryData' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deployment' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'id' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageCVECountBySeverity' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'FragmentSpread',
                                                name: {
                                                    kind: 'Name',
                                                    value: 'ResourceCountsByCVESeverityAndStatus',
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ResourceCountsByCVESeverityAndStatus' },
            typeCondition: {
                kind: 'NamedType',
                name: { kind: 'Name', value: 'ResourceCountByCVESeverity' },
            },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'low' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'moderate' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'important' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'critical' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetDeploymentSummaryDataQuery, GetDeploymentSummaryDataQueryVariables>;
export const GetCvesForDeploymentDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getCvesForDeployment' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'pagination' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'Pagination' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deployment' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'id' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageVulnerabilityCount' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                },
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'DeploymentWithVulnerabilities' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageMetadataContext' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'name' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'registry' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'remote' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'metadata' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'v1' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'layers' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'instruction',
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'value' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentComponentVulnerabilities' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'ImageComponent' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'version' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'location' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'source' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'layerIndex' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageVulnerabilities' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnerabilityId' },
                                    name: { kind: 'Name', value: 'id' },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'severity' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'cvss' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'scoreVersion' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixedByVersion' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'discoveredAtImage' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentWithVulnerabilities' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Deployment' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'images' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImageMetadataContext' },
                                },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageVulnerabilities' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnerabilityId' },
                                    name: { kind: 'Name', value: 'id' },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'cve' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'summary' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'images' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                alias: { kind: 'Name', value: 'imageId' },
                                                name: { kind: 'Name', value: 'id' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'imageComponents' },
                                                arguments: [
                                                    {
                                                        kind: 'Argument',
                                                        name: { kind: 'Name', value: 'query' },
                                                        value: {
                                                            kind: 'Variable',
                                                            name: { kind: 'Name', value: 'query' },
                                                        },
                                                    },
                                                ],
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'FragmentSpread',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'DeploymentComponentVulnerabilities',
                                                            },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetCvesForDeploymentQuery, GetCvesForDeploymentQueryVariables>;
export const GetImageDetailsDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getImageDetails' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'image' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'id' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'name' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'registry' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'remote' },
                                            },
                                            { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                                        ],
                                    },
                                },
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImageDetails' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageDetails' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'deploymentCount' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'operatingSystem' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'metadata' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'v1' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'created' },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'dataSource' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [{ kind: 'Field', name: { kind: 'Name', value: 'name' } }],
                        },
                    },
                    { kind: 'Field', name: { kind: 'Name', value: 'scanTime' } },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetImageDetailsQuery, GetImageDetailsQueryVariables>;
export const GetImageResourcesDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getImageResources' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'pagination' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'Pagination' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'image' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'id' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'DeploymentResources' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentResources' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deploymentCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deployments' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'clusterName' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'namespace' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'created' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetImageResourcesQuery, GetImageResourcesQueryVariables>;
export const GetCvEsForImageDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getCVEsForImage' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'ID' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'pagination' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'Pagination' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'image' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'id' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'id' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImageMetadataContext' },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageCVECountBySeverity' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'FragmentSpread',
                                                name: {
                                                    kind: 'Name',
                                                    value: 'ResourceCountsByCVESeverityAndStatus',
                                                },
                                            },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageVulnerabilities' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'pagination' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'pagination' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'FragmentSpread',
                                                name: {
                                                    kind: 'Name',
                                                    value: 'ImageVulnerabilityFields',
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageComponentVulnerabilities' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'ImageComponent' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'version' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'location' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'source' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'layerIndex' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageVulnerabilities' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnerabilityId' },
                                    name: { kind: 'Name', value: 'id' },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'severity' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixedByVersion' } },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageMetadataContext' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'name' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'registry' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'remote' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'metadata' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'v1' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'layers' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'instruction',
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'value' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ResourceCountsByCVESeverityAndStatus' },
            typeCondition: {
                kind: 'NamedType',
                name: { kind: 'Name', value: 'ResourceCountByCVESeverity' },
            },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'low' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'moderate' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'important' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'critical' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageVulnerabilityFields' },
            typeCondition: {
                kind: 'NamedType',
                name: { kind: 'Name', value: 'ImageVulnerability' },
            },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'severity' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'cve' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'summary' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'cvss' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'scoreVersion' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'discoveredAtImage' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageComponents' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImageComponentVulnerabilities' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetCvEsForImageQuery, GetCvEsForImageQueryVariables>;
export const GetImageCveMetadataDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getImageCveMetadata' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'cve' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageCVE' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'cve' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'cve' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImageCVEMetadata' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageCVEMetadata' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'ImageCVECore' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'cve' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'firstDiscoveredInSystem' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'distroTuples' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'summary' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'link' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'operatingSystem' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetImageCveMetadataQuery, GetImageCveMetadataQueryVariables>;
export const GetImageCveSummaryDataDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getImageCveSummaryData' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'cve' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                    },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: {
                        kind: 'NonNullType',
                        type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                    },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'totalImageCount' },
                        name: { kind: 'Name', value: 'imageCount' },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deploymentCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageCVE' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'cve' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'cve' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'subfieldScopeQuery' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'cve' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'affectedImageCount' },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'affectedImageCountBySeverity' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'FragmentSpread',
                                                name: {
                                                    kind: 'Name',
                                                    value: 'ResourceCountsByCVESeverityAndStatus',
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ResourceCountsByCVESeverityAndStatus' },
            typeCondition: {
                kind: 'NamedType',
                name: { kind: 'Name', value: 'ResourceCountByCVESeverity' },
            },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'low' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'moderate' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'important' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'critical' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'total' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixable' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetImageCveSummaryDataQuery, GetImageCveSummaryDataQueryVariables>;
export const GetImagesForCveDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getImagesForCVE' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'pagination' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'Pagination' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'images' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImagesForCVE' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageMetadataContext' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'name' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'registry' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'remote' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'metadata' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'v1' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'layers' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'instruction',
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'value' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageComponentVulnerabilities' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'ImageComponent' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'version' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'location' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'source' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'layerIndex' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageVulnerabilities' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnerabilityId' },
                                    name: { kind: 'Name', value: 'id' },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'severity' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixedByVersion' } },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImagesForCVE' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'FragmentSpread',
                        name: { kind: 'Name', value: 'ImageMetadataContext' },
                    },
                    { kind: 'Field', name: { kind: 'Name', value: 'operatingSystem' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'watchStatus' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'scanTime' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageComponents' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageVulnerabilities' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'cvss' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'scoreVersion' },
                                            },
                                        ],
                                    },
                                },
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImageComponentVulnerabilities' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetImagesForCveQuery, GetImagesForCveQueryVariables>;
export const GetDeploymentsForCveDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getDeploymentsForCVE' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'pagination' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'Pagination' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: {
                        kind: 'Variable',
                        name: { kind: 'Name', value: 'lowImageCountQuery' },
                    },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: {
                        kind: 'Variable',
                        name: { kind: 'Name', value: 'moderateImageCountQuery' },
                    },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: {
                        kind: 'Variable',
                        name: { kind: 'Name', value: 'importantImageCountQuery' },
                    },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: {
                        kind: 'Variable',
                        name: { kind: 'Name', value: 'criticalImageCountQuery' },
                    },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deployments' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'DeploymentsForCVE' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'ImageMetadataContext' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Image' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'name' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'registry' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'remote' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                            ],
                        },
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'metadata' },
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'v1' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'layers' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'instruction',
                                                            },
                                                        },
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'value' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentComponentVulnerabilities' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'ImageComponent' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'version' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'location' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'source' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'layerIndex' } },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageVulnerabilities' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'Field',
                                    alias: { kind: 'Name', value: 'vulnerabilityId' },
                                    name: { kind: 'Name', value: 'id' },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'severity' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'cvss' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'scoreVersion' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'fixedByVersion' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'discoveredAtImage' },
                                },
                            ],
                        },
                    },
                ],
            },
        },
        {
            kind: 'FragmentDefinition',
            name: { kind: 'Name', value: 'DeploymentsForCVE' },
            typeCondition: { kind: 'NamedType', name: { kind: 'Name', value: 'Deployment' } },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'namespace' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'clusterName' } },
                    { kind: 'Field', name: { kind: 'Name', value: 'created' } },
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'lowImageCount' },
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'lowImageCountQuery' },
                                },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'moderateImageCount' },
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'moderateImageCountQuery' },
                                },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'importantImageCount' },
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'importantImageCountQuery' },
                                },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'criticalImageCount' },
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'criticalImageCountQuery' },
                                },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'images' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                {
                                    kind: 'FragmentSpread',
                                    name: { kind: 'Name', value: 'ImageMetadataContext' },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageComponents' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'FragmentSpread',
                                                name: {
                                                    kind: 'Name',
                                                    value: 'DeploymentComponentVulnerabilities',
                                                },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetDeploymentsForCveQuery, GetDeploymentsForCveQueryVariables>;
export const GetImageCveListDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getImageCVEList' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'pagination' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'Pagination' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageCVEs' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'cve' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'affectedImageCountBySeverity' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'critical' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                    ],
                                                },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'important' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                    ],
                                                },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'moderate' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                    ],
                                                },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'low' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'topCVSS' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'affectedImageCount' },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'firstDiscoveredInSystem' },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'distroTuples' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'summary' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'operatingSystem' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'cvss' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'scoreVersion' },
                                            },
                                        ],
                                    },
                                },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetImageCveListQuery, GetImageCveListQueryVariables>;
export const GetUnfilteredImageCountDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getUnfilteredImageCount' },
            selectionSet: {
                kind: 'SelectionSet',
                selections: [{ kind: 'Field', name: { kind: 'Name', value: 'imageCount' } }],
            },
        },
    ],
} as unknown as DocumentNode<GetUnfilteredImageCountQuery, GetUnfilteredImageCountQueryVariables>;
export const GetDeploymentListDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getDeploymentList' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'pagination' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'Pagination' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deployments' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'name' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageCVECountBySeverity' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'critical' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                    ],
                                                },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'important' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                    ],
                                                },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'moderate' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                    ],
                                                },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'low' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'clusterName' } },
                                { kind: 'Field', name: { kind: 'Name', value: 'namespace' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageCount' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'created' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetDeploymentListQuery, GetDeploymentListQueryVariables>;
export const GetImageListDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getImageList' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'pagination' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'Pagination' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'images' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'pagination' },
                                value: {
                                    kind: 'Variable',
                                    name: { kind: 'Name', value: 'pagination' },
                                },
                            },
                        ],
                        selectionSet: {
                            kind: 'SelectionSet',
                            selections: [
                                { kind: 'Field', name: { kind: 'Name', value: 'id' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'name' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'registry' },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'remote' },
                                            },
                                            { kind: 'Field', name: { kind: 'Name', value: 'tag' } },
                                        ],
                                    },
                                },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'imageCVECountBySeverity' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'critical' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                    ],
                                                },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'important' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                    ],
                                                },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'moderate' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                    ],
                                                },
                                            },
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'low' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: { kind: 'Name', value: 'total' },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'operatingSystem' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'deploymentCount' },
                                    arguments: [
                                        {
                                            kind: 'Argument',
                                            name: { kind: 'Name', value: 'query' },
                                            value: {
                                                kind: 'Variable',
                                                name: { kind: 'Name', value: 'query' },
                                            },
                                        },
                                    ],
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'watchStatus' } },
                                {
                                    kind: 'Field',
                                    name: { kind: 'Name', value: 'metadata' },
                                    selectionSet: {
                                        kind: 'SelectionSet',
                                        selections: [
                                            {
                                                kind: 'Field',
                                                name: { kind: 'Name', value: 'v1' },
                                                selectionSet: {
                                                    kind: 'SelectionSet',
                                                    selections: [
                                                        {
                                                            kind: 'Field',
                                                            name: {
                                                                kind: 'Name',
                                                                value: 'created',
                                                            },
                                                        },
                                                    ],
                                                },
                                            },
                                        ],
                                    },
                                },
                                { kind: 'Field', name: { kind: 'Name', value: 'scanTime' } },
                            ],
                        },
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetImageListQuery, GetImageListQueryVariables>;
export const GetEntityTypeCountsDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getEntityTypeCounts' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'deploymentCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                    },
                    {
                        kind: 'Field',
                        name: { kind: 'Name', value: 'imageCVECount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetEntityTypeCountsQuery, GetEntityTypeCountsQueryVariables>;
export const GetDeploymentCountDocument = {
    kind: 'Document',
    definitions: [
        {
            kind: 'OperationDefinition',
            operation: 'query',
            name: { kind: 'Name', value: 'getDeploymentCount' },
            variableDefinitions: [
                {
                    kind: 'VariableDefinition',
                    variable: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                    type: { kind: 'NamedType', name: { kind: 'Name', value: 'String' } },
                },
            ],
            selectionSet: {
                kind: 'SelectionSet',
                selections: [
                    {
                        kind: 'Field',
                        alias: { kind: 'Name', value: 'count' },
                        name: { kind: 'Name', value: 'deploymentCount' },
                        arguments: [
                            {
                                kind: 'Argument',
                                name: { kind: 'Name', value: 'query' },
                                value: { kind: 'Variable', name: { kind: 'Name', value: 'query' } },
                            },
                        ],
                    },
                ],
            },
        },
    ],
} as unknown as DocumentNode<GetDeploymentCountQuery, GetDeploymentCountQueryVariables>;
