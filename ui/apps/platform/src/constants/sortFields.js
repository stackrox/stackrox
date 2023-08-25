import entityTypes from 'constants/entityTypes';

export const imageSortFields = {
    CVE_COUNT: 'CVE Count',
    TOP_CVSS: 'Image Top CVSS',
    CLUSTER: 'Cluster',
    COMPONENT: 'Component',
    COMPONENT_COUNT: 'Component Count',
    COMPONENT_VERSION: 'Component Version',
    DEPLOYMENT: 'Deployment',
    DEPLOYMENT_COUNT: 'Deployment Count',
    DOCKERFILE_INSTRUCTION_KEYWORD: 'Dockerfile Instruction Keyword',
    DOCKERFILE_INSTRUCTION_VALUE: 'Dockerfile Instruction Value',
    FIXABLE_CVE_COUNT: 'Fixable CVE Count',
    FIXED_BY: 'Fixed By',
    NAME: 'Image',
    COMMAND: 'Image Command',
    CREATED_TIME: 'Image Created Time',
    ENTRYPOINT: 'Image Entrypoint',
    IMAGE_STATUS: 'Image Status',
    IMAGE_OS: 'Image OS',
    PRIORITY: 'Image Risk Priority',
    REGISTRY: 'Image Registry',
    REMOTE: 'Image Remote',
    SCAN_TIME: 'Image Scan Time',
    TAG: 'Image Tag',
    USER: 'Image User',
    VOLUMES: 'Image Volumes',
    LABEL: 'Label',
    NAMESPACE: 'Namespace',
    NAMESPACE_ID: 'Namespace ID',
};

/**
 * derived from search categories for images,
 *   ["CVE","CVE Count","CVSS","Cluster","Component","Component Count","Component Version","Deployment","Dockerfile Instruction Keyword","Dockerfile Instruction Value","Fixable CVE Count","Fixed By","Image","Image Command","Image Created Time","Image Entrypoint","Image Registry","Image Remote","Image Scan Time","Image Tag","Image User","Image Volumes","Label","Namespace","Namespace ID"]
 *   ???
 *
 * plus component-specific table columns
 *   Component, CVE Count, Top CVSS, Deployments, Images, Priority
 */
export const componentSortFields = {
    ACTIVE: 'Active',
    COMPONENT: 'Component',
    CVE_COUNT: 'CVE Count',
    TOP_CVSS: 'Component Top CVSS',
    SOURCE: 'Component Source',
    LOCATION: 'Component Location',
    IMAGE_COUNT: 'Image Count',
    DEPLOYMENT_COUNT: 'Deployment Count',
    PRIORITY: 'Component Risk Priority',
    FIXEDIN: 'Component Fixed By', // This field does not exist. However, seems like every column has to follow a template.
    OPERATING_SYSTEM: 'Operating System',
};

/**
 * derived from search categories for images,
 *   ["CVE","CVE Count","CVSS","Cluster","Component","Component Count","Component Version","Deployment","Dockerfile Instruction Keyword","Dockerfile Instruction Value","Fixable CVE Count","Fixed By","Image","Image Command","Image Created Time","Image Entrypoint","Image Registry","Image Remote","Image Scan Time","Image Tag","Image User","Image Volumes","Label","Namespace","Namespace ID"]
 *   ???
 *
 * plus cve-specific table columns
 *   CVE, CVSS Score, Fixable, Env. Impact, Impact Score, Deployments, Images, Components, Scanned, Published
 */
export const cveSortFields = {
    ACTIVE: 'Active',
    CVE: 'CVE',
    CVE_TYPE: 'CVE Type',
    CVSS_SCORE: 'CVSS',
    SUPPRESSED: 'CVE Snoozed',
    SEVERITY: 'Severity',
    FIXABLE: 'Fixable',
    FIXEDIN: 'Fixed By',
    ENV_IMPACT: 'Env. Impact',
    IMPACT_SCORE: 'Impact Score',
    DEPLOYMENT_COUNT: 'Deployment Count',
    IMAGE_COUNT: 'Image Count',
    COMPONENT_COUNT: 'Component Count',
    SCANNED: 'Last Scanned',
    CVE_CREATED_TIME: 'CVE Created Time',
    CVE_DISCOVERED_AT_IMAGE_TIME: 'CVE Discovered at Image Time',
    IMAGE_SCAN_TIME: 'Image Scan Time',
    PUBLISHED: 'CVE Published On',
    OPERATING_SYSTEM: 'Operating System',
};

/**
 * derived from search categories for clusters,
 *   "Cluster"
 *
 * plus cluster-specific table columns
 *   CVEs, K8sVersion, Namespaces, Deployments, Policies, Policy Status, Latest Violation, Risk Priority
 */
export const clusterSortFields = {
    CVE_COUNT: 'CVE Count',
    CLUSTER: 'Cluster',
    DEPLOYMENT_COUNT: 'Deployment Count',
    K8SVERSION: 'K8S Version',
    LATEST_VIOLATION: 'Violation Time',
    NAME: 'Cluster',
    NAMESPACE_COUNT: 'Namespace Count',
    POLICY_COUNT: 'Policy Count',
    POLICY_STATUS: 'Policy Status',
    PRIORITY: 'Cluster Risk Priority',
};

/**
 * derived from search categories for namespaces,
 *   "Cluster", "Label", "Namespace", "Namespace ID"
 *
 * plus namespace-specific table columns
 *   CVEs, Deployments, Images, Policies, Policy Status, Latest Violation, Priority
 */
export const namespaceSortFields = {
    CVE_COUNT: 'CVE Count',
    CLUSTER: 'Cluster',
    DEPLOYMENT_COUNT: 'Deployment Count',
    IMAGES: 'Images',
    LATEST_VIOLATION: 'Violation Time',
    NAMESPACE: 'Namespace',
    NAME: 'Namespace',
    POLICY_COUNT: 'Policy Count',
    POLICY_STATUS: 'Policy Status',
    PRIORITY: 'Namespace Risk Priority',
};

export const nodeSortFields = {
    CLUSTER: 'Cluster',
    CONTAINER_RUNTIME: 'Container Runtime',
    NODE: 'Node',
    NODE_JOIN_TIME: 'Node Join Time',
    OPERATING_SYSTEM: 'Operating System',
    CVE_COUNT: 'CVE Count',
    PRIORITY: 'Node Risk Priority',
    TOP_CVSS: 'Node Top CVSS',
    IMAGE_OS: 'Image OS',
    SCAN_TIME: 'Node Scan Time',
};

/**
 * added in order to use backend pagination on Config Mgmt Secrets list page
 */
export const secretSortFields = {
    SECRET: 'Secret',
    CREATED: 'Created Time',
    CLUSTER: 'Cluster',
};

/**
 * added in order to use backend pagination on Config Mgmt Roles list page
 */
export const roleSortFields = {
    ROLE: 'Role',
    CLUSTER: 'Cluster',
};

/**
 * added in order to use backend pagination on Config Mgmt Service Account list page
 */
export const serviceAccountSortFields = {
    SERVCE_ACCOUNT: 'Service Account',
    CLUSTER: 'Cluster',
    NAMESPACE: 'Namespace',
};

/**
 * added in order to use backend pagination on Config Mgmt Subject list page
 */
export const subjectSortFields = {
    SUBJECT: 'Subject',
    SUBJECT_KIND: 'Subject Kind',
};

/**
 * added in order to use backend pagination on Config Mgmt Nodes list page
 *
 * completely derived from trial-and-error
 */

/**
 * derived from search categories for policies,
 *   "Category", "Description", "Disabled", "Enforcement", "Lifecycle Stage", "Policy", "Severity"
 *
 * plus policy-specific table columns
 */
export const policySortFields = {
    CATEGORY: 'Category',
    DEPLOYMENT_COUNT: 'Deployment Count',
    DESCRIPTION: 'Description',
    DISABLED: 'Disabled',
    ENFORCEMENT: 'SORT_Enforcement',
    LAST_UPDATED: 'Policy Last Updated',
    LATEST_VIOLATION: 'Violation Time',
    LIFECYCLE_STAGE: 'Lifecycle Stage',
    POLICY: 'Policy',
    POLICY_STATUS: 'Policy Status',
    SEVERITY: 'Severity',
};

export const deploymentSortFields = {
    ADD_CAPABILITIES: 'Add Capabilities',
    ANNOTATION: 'Annotation',
    CPU_CORES_LIMIT: 'CPU Cores Limit',
    CPU_CORES_REQUEST: 'CPU Cores Request',
    CVE: 'CVE',
    CVE_COUNT: 'CVE Count',
    LATEST_VIOLATION: 'Violation Time',
    CVSS: 'CVSS',
    CLUSTER: 'Cluster',
    COMPONENT: 'Component',
    COMPONENT_COUNT: 'Component Count',
    COMPONENT_VERSION: 'Component Version',
    DEPLOYMENT: 'Deployment',
    DEPLOYMENT_TYPE: 'Deployment Type',
    DOCKERFILE_INSTRUCTION_KEYWORD: 'Dockerfile Instruction Keyword',
    DOCKERFILE_INSTRUCITON_VALUE: 'Dockerfile Instruction Value',
    DROP_CAPABILITIES: 'Drop Capabilities',
    ENVIRONMENT_KEY: 'Environment Key',
    ENVIRONMENT_VALUE: 'Environment Value',
    ENVIRONMENT_VARIABLE_SOURCE: 'Environment Variable Source',
    EXPOSED_NODE_PORT: 'Exposed Node Port',
    EXPOSING_SERVICE: 'Exposing Service',
    EXPOSING_SERVICE_PORT: 'Exposing Service Port',
    EXPOSURE_LEVEL: 'Exposure Level',
    EXTERNAL_HOSTNAME: 'External Hostname',
    EXTERNAL_IP: 'External IP',
    FIXABLE_CVE_COUNT: 'Fixable CVE Count',
    FIXED_BY: 'Fixed By',
    IMAGE: 'Image',
    IMAGES: 'Images',
    IMAGE_COUNT: 'Image Count',
    IMAGE_COMMAND: 'Image Command',
    IMAGE_CREATED_TIME: 'Image Created Time',
    IMAGE_ENTRYPOINT: 'Image Entrypoint',
    IMAGE_PULL_SECRET: 'Image Pull Secret',
    IMAGE_REGISTRY: 'Image Registry',
    IMAGE_REMOTE: 'Image Remote',
    IMAGE_SCAN_TIME: 'Image Scan Time',
    IMAGE_TAG: 'Image Tag',
    IMAGE_USER: 'Image User',
    IMAGE_VOLUMES: 'Image Volumes',
    LABEL: 'Label',
    MAX_EXPOSURE_LEVEL: 'Max Exposure Level',
    MEMORY_LIMIT: 'Memory Limit (MB)',
    MEMORY_REQUEST: 'Memory Request (MB)',
    NAME: 'Deployment',
    NAMESPACE: 'Namespace',
    NAMESPACE_ID: 'Namespace ID',
    POD_LABEL: 'Pod Label',
    POLICY_COUNT: 'Policy Count',
    POLICY_STATUS: 'Policy Status',
    PORT: 'Port',
    PORT_PROTOCOL: 'Port Protocol',
    PRIORITY: 'Deployment Risk Priority',
    PRIVILEGED: 'Privileged',
    PROCESS_ANCESTOR: 'Process Ancestor',
    PROCESS_ARGUMENTS: 'Process Arguments',
    PROCESS_NAME: 'Process Name',
    PROCESS_PATH: 'Process Path',
    PROCESS_UID: 'Process UID',
    READ_ONLY_ROOT_FILESYSTEM: 'Read Only Root Filesystem',
    SECRET: 'Secret',
    SECRET_PATH: 'Secret Path',
    SERVICE_ACCOUNT: 'Service Account',
    TOLERATION_KEY: 'Toleration Key',
    TOLERATION_VALUE: 'Toleration Value',
    VOLUME_DESTINATION: 'Volume Destination',
    VOLUME_NAME: 'Volume Name',
    VOLUME_READONLY: 'Volume ReadOnly',
    VOLUME_SOURCE: 'Volume Source',
    VOLUME_TYPE: 'Volume Type',
};

// rollup object export
//   all the sort fields combined in one big object
export const entitySortFieldsMap = {
    [entityTypes.CLUSTER]: clusterSortFields,
    [entityTypes.NAMESPACE]: namespaceSortFields,
    [entityTypes.DEPLOYMENT]: deploymentSortFields,
    [entityTypes.IMAGE]: imageSortFields,
    [entityTypes.COMPONENT]: componentSortFields,
    [entityTypes.CVE]: cveSortFields,
    [entityTypes.POLICY]: policySortFields,
    [entityTypes.NODE]: nodeSortFields,
    [entityTypes.NODE_CVE]: cveSortFields,
    [entityTypes.NODE_COMPONENT]: componentSortFields,
    [entityTypes.IMAGE_CVE]: cveSortFields,
    [entityTypes.IMAGE_COMPONENT]: componentSortFields,
};
