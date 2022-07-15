import entityTypes from 'constants/entityTypes';

export const WIDGET_PAGINATION_START_OFFSET = 0;

export const OVERVIEW_LIMIT = 5;

export const DASHBOARD_LIMIT = 8;

export const LIST_PAGE_SIZE = 25;

export const defaultCountKeyMap = {
    [entityTypes.COMPONENT]: 'componentCount',
    [entityTypes.NODE_COMPONENT]: 'componentCount: nodeComponentCount',
    [entityTypes.IMAGE_COMPONENT]: 'componentCount: imageComponentCount',
    [entityTypes.CVE]: 'vulnCount',
    [entityTypes.IMAGE_CVE]: 'vulnCount: imageVulnerabilityCount',
    [entityTypes.NODE_CVE]: 'vulnCount: nodeVulnerabilityCount',
    [entityTypes.CLUSTER_CVE]: 'vulnCount: clusterVulnerabilityCount',
    [entityTypes.K8S_CVE]: 'vulnCount: k8sVulnCount',
    [entityTypes.DEPLOYMENT]: 'deploymentCount',
    [entityTypes.NAMESPACE]: 'namespaceCount',
    [entityTypes.NODE]: 'nodeCount',
    [entityTypes.IMAGE]: 'imageCount',
    [entityTypes.POLICY]: 'policyCount',
    [entityTypes.SECRET]: 'secretCount',
    [entityTypes.SUBJECT]: 'subjectCount',
    [entityTypes.SERVICE_ACCOUNT]: 'serviceAccountCount',
    [entityTypes.ROLE]: 'k8sRoleCount',
};
