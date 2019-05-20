export const resourceTypes = {
    NAMESPACE: 'NAMESPACE',
    CLUSTER: 'CLUSTER',
    NODE: 'NODE',
    DEPLOYMENT: 'DEPLOYMENT',
    NETWORK_POLICY: 'NETWORK_POLICY',
    SECRET: 'SECRET'
};

export const standardEntityTypes = {
    CONTROL: 'CONTROL',
    CATEGORY: 'CATEGORY',
    STANDARD: 'STANDARD'
};

export const standardTypes = {
    PCI_DSS_3_2: 'PCI_DSS_3_2',
    NIST_800_190: 'NIST_800_190',
    HIPAA_164: 'HIPAA_164',
    CIS_Kubernetes_v1_2_0: 'CIS_Kubernetes_v1_2_0',
    CIS_Docker_v1_1_0: 'CIS_Docker_v1_1_0'
};

export const standardBaseTypes = {
    [standardTypes.PCI_DSS_3_2]: 'PCI',
    [standardTypes.NIST_800_190]: 'NIST',
    [standardTypes.HIPAA_164]: 'HIPAA',
    [standardTypes.CIS_Docker_v1_1_0]: 'CIS Docker',
    [standardTypes.CIS_Kubernetes_v1_2_0]: 'CIS K8s'
};

export const searchCategories = {
    NAMESPACE: 'NAMESPACES',
    NODE: 'NODES',
    CLUSTER: 'CLUSTERS',
    CONTROL: 'COMPLIANCE',
    DEPLOYMENT: 'DEPLOYMENTS',
    SECRET: 'SECRETS'
};

export default {
    ...resourceTypes,
    ...standardTypes,
    ...standardEntityTypes
};
