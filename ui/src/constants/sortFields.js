export const imageSortFields = {
    CVE: 'CVE',
    CVE_COUNT: 'CVE Count',
    CVSS: 'CVSS',
    CLUSTER: 'Cluster',
    COMPONENT: 'Component',
    COMPONENT_COUNT: 'Component Count',
    COMPONENT_VERSION: 'Component Version',
    DEPLOYMENT: 'Deployment',
    DOCKERFILE_INSTRUCTION_KEYWORD: 'Dockerfile Instruction Keyword',
    DOCKERFILE_INSTRUCTION_VALUE: 'Dockerfile Instruction Value',
    FIXABLE_CVE_COUNT: 'Fixable CVE Count',
    FIXED_BY: 'Fixed By',
    NAME: 'Image',
    COMMAND: 'Image Command',
    CREATED_TIME: 'Image Created Time',
    ENTRYPOINT: 'Image Entrypoint',
    REGISTRY: 'Image Registry',
    REMOTE: 'Image Remote',
    SCAN_TIME: 'Image Scan Time',
    TAG: 'Image Tag',
    USER: 'Image User',
    VOLUMES: 'Image Volumes',
    LABEL: 'Label',
    NAMESPACE: 'Namespace',
    NAMESPACE_ID: 'Namespace ID'
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
    COMPONENT: 'Component',
    CVE_COUNT: 'CVE Count',
    TOP_CVSS: 'Top CVSS',
    IMAGES: 'Images',
    DEPLOYMENTS: 'Deployments',
    PRIORITY: 'Priority'
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
    CVE: 'CVE',
    CVSS_SCORE: 'CVSS Score',
    FIXABLE: 'Fixable',
    ENV_IMPACT: 'Env. Impact',
    IMPACT_SCORE: 'Impact Score',
    DEPLOYMENTS: 'Deployments',
    IMAGES: 'Images',
    COMPONENTS: 'Components',
    SCANNED: 'Scanned',
    PUBLISHED: 'Published'
};

/**
 * derived from search categories for clusters,
 *   "Cluster"
 *
 * plus cluster-specific table columns
 *   CVEs, K8sVersion, Namespaces, Deployments, Policies, Policy Status, Latest Violation, Risk Priority
 */
export const clusterSortFields = {
    CVES: 'CVES',
    CLUSTER: 'Cluster',
    DEPLOYMENTS: 'Deployments',
    K8SVERSION: 'K8S Version',
    LATEST_VIOLATION: 'Latest Violation',
    NAMESPACE: 'Namespace',
    POLICIES: 'Policies',
    POLICY_STATUS: 'Policy Status',
    PRIORITY: 'Priority'
};

/**
 * derived from search categories for namespaces,
 *   "Cluster", "Label", "Namespace", "Namespace ID"
 *
 * plus namespace-specific table columns
 *   CVEs, Deployments, Images, Policies, Policy Status, Latest Violation, Priority
 */
export const namespaceSortFields = {
    CVES: 'CVES',
    CLUSTER: 'Cluster',
    DEPLOYMENTS: 'Deployments',
    IMAGES: 'Images',
    LATEST_VIOLATION: 'Latest Violation',
    NAMESPACE: 'Namespace',
    POLICIES: 'Policies',
    POLICY_STATUS: 'Policy Status',
    PRIORITY: 'Priority'
};

/**
 * derived from search categories for policies,
 *   "Category", "Description", "Disabled", "Enforcement", "Lifecycle Stage", "Policy", "Severity"
 *
 * plus policy-specific table columns
 */
export const policySortFields = {
    CATEGORY: 'Category',
    DEPLOYMENTS: 'Deployments',
    DESCRIPTION: 'Description',
    DISABLED: 'Disabled',
    ENFORCEMENT: 'Enforcement',
    LAST_UPDATED: 'Last Updated',
    LATEST_VIOLATION: 'Latest Violation',
    LIFECYCLE_STAGE: 'Lifecycle Stage',
    POLICY: 'Policy',
    POLICY_STATUS: 'Policy Status',
    SEVERITY: 'Severity'
};

export const deploymentSortFields = {
    ADD_CAPABILITIES: 'Add Capabilities',
    ANNOTATION: 'Annotation',
    CPU_CORES_LIMIT: 'CPU Cores Limit',
    CPU_CORES_REQUEST: 'CPU Cores Request',
    CVE: 'CVE',
    CVE_COUNT: 'CVE Count',
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
    NAMESPACE: 'Namespace',
    NAMESPACE_ID: 'Namespace ID',
    POD_LABEL: 'Pod Label',
    PORT: 'Port',
    PORT_PROTOCOL: 'Port Protocol',
    PRIORITY: 'Priority',
    PRIVILAGED: 'Privileged',
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
    VOLUME_TYPE: 'Volume Type'
};
