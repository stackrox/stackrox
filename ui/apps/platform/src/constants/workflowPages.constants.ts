export const WIDGET_PAGINATION_START_OFFSET = 0;

export const OVERVIEW_LIMIT = 5;

export const DASHBOARD_LIMIT = 8;

export const LIST_PAGE_SIZE = 25;

export const defaultCountKeyMap = {
    CLUSTER: 'clusterCount',

    COMPONENT: 'componentCount',
    NODE_COMPONENT: 'nodeComponentCount',
    IMAGE_COMPONENT: 'imageComponentCount',

    CVE: 'vulnCount',
    CLUSTER_CVE: 'clusterVulnerabilityCount',
    IMAGE_CVE: 'imageVulnerabilityCount',
    K8S_CVE: 'vulnCount: k8sVulnCount', // was broken, is it used?
    NODE_CVE: 'nodeVulnerabilityCount',

    DEPLOYMENT: 'deploymentCount',
    IMAGE: 'imageCount',
    NAMESPACE: 'namespaceCount',
    NODE: 'nodeCount',
    POLICY: 'policyCount',
    ROLE: 'k8sRoleCount',
    SECRET: 'secretCount',
    SERVICE_ACCOUNT: 'serviceAccountCount',
    SUBJECT: 'subjectCount',
};
