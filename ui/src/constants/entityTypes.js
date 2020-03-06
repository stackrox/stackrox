export const resourceTypes = {
    NAMESPACE: 'NAMESPACE',
    CLUSTER: 'CLUSTER',
    NODE: 'NODE',
    DEPLOYMENT: 'DEPLOYMENT',
    NETWORK_POLICY: 'NETWORK_POLICY',
    SECRET: 'SECRET',
    IMAGE: 'IMAGE',
    COMPONENT: 'COMPONENT',
    CVE: 'CVE',
    POLICY: 'POLICY',
    CONTROL: 'CONTROL'
};

export const rbacConfigTypes = {
    SUBJECT: 'SUBJECT',
    SERVICE_ACCOUNT: 'SERVICE_ACCOUNT',
    ROLE: 'ROLE'
};

export const standardEntityTypes = {
    CONTROL: 'CONTROL',
    CATEGORY: 'CATEGORY',
    STANDARD: 'STANDARD',
    CHECK: 'CHECK'
};

export const standardTypes = {
    PCI_DSS_3_2: 'PCI_DSS_3_2',
    NIST_800_190: 'NIST_800_190',
    NIST_SP_800_53_Rev_4: 'NIST_SP_800_53_Rev_4',
    HIPAA_164: 'HIPAA_164',
    CIS_Kubernetes_v1_5: 'CIS_Kubernetes_v1_5',
    CIS_Docker_v1_1_0: 'CIS_Docker_v1_1_0',
    CIS_Docker_v1_2_0: 'CIS_Docker_v1_2_0'
};

export const standardBaseTypes = {
    [standardTypes.PCI_DSS_3_2]: 'PCI',
    [standardTypes.NIST_800_190]: 'NIST SP 800-190',
    [standardTypes.NIST_SP_800_53_Rev_4]: 'NIST SP 800-53',
    [standardTypes.HIPAA_164]: 'HIPAA',
    [standardTypes.CIS_Docker_v1_1_0]: 'CIS Docker',
    [standardTypes.CIS_Docker_v1_2_0]: 'CIS Docker',
    [standardTypes.CIS_Kubernetes_v1_5]: 'CIS K8s'
};

// resourceTypeToApplicableStandards maps a resource type to the standards that apply to it.
// This is required because not all standards have node and deployment checks.
// (Although every standard will be in the cluster sub-list, in the current data model.)
export const resourceTypeToApplicableStandards = {
    [resourceTypes.CLUSTER]: [
        standardTypes.CIS_Docker_v1_2_0,
        standardTypes.CIS_Kubernetes_v1_5,
        standardTypes.HIPAA_164,
        standardTypes.NIST_800_190,
        standardTypes.NIST_SP_800_53_Rev_4,
        standardTypes.PCI_DSS_3_2
    ],
    [resourceTypes.NODE]: [
        standardTypes.CIS_Docker_v1_2_0,
        standardTypes.CIS_Kubernetes_v1_5,
        standardTypes.NIST_800_190
    ],
    [resourceTypes.NAMESPACE]: [
        standardTypes.HIPAA_164,
        standardTypes.NIST_800_190,
        standardTypes.NIST_SP_800_53_Rev_4,
        standardTypes.PCI_DSS_3_2
    ],
    [resourceTypes.DEPLOYMENT]: [
        standardTypes.HIPAA_164,
        standardTypes.NIST_800_190,
        standardTypes.NIST_SP_800_53_Rev_4,
        standardTypes.PCI_DSS_3_2
    ]
};

export const searchCategories = {
    NAMESPACE: 'NAMESPACES',
    NODE: 'NODES',
    CLUSTER: 'CLUSTERS',
    CONTROL: 'COMPLIANCE',
    CVE: 'VULNERABILITIES',
    COMPONENT: 'IMAGE_COMPONENTS',
    DEPLOYMENT: 'DEPLOYMENTS',
    SECRET: 'SECRETS',
    POLICY: 'POLICIES',
    IMAGE: 'IMAGES',
    RISK: 'RISKS',
    ROLE: 'ROLES',
    SERVICE_ACCOUNT: 'SERVICE_ACCOUNTS',
    SUBJECT: 'SUBJECTS'
};

export default {
    ...resourceTypes,
    ...standardTypes,
    ...standardEntityTypes,
    ...rbacConfigTypes
};
