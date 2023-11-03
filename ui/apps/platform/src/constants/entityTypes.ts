import { SearchCategory } from 'services/SearchService';

export type ResourceType =
    | 'NAMESPACE'
    | 'CLUSTER'
    | 'NODE'
    | 'DEPLOYMENT'
    | 'NETWORK_POLICY'
    | 'SECRET'
    | 'IMAGE'
    | 'COMPONENT'
    | 'NODE_COMPONENT'
    | 'IMAGE_COMPONENT'
    | 'CVE'
    | 'IMAGE_CVE'
    | 'NODE_CVE'
    | 'CLUSTER_CVE'
    | 'POLICY'
    | 'CONTROL';

export const resourceTypes: Record<ResourceType, ResourceType> = {
    NAMESPACE: 'NAMESPACE',
    CLUSTER: 'CLUSTER',
    NODE: 'NODE',
    DEPLOYMENT: 'DEPLOYMENT',
    NETWORK_POLICY: 'NETWORK_POLICY',
    SECRET: 'SECRET',
    IMAGE: 'IMAGE',
    COMPONENT: 'COMPONENT',
    NODE_COMPONENT: 'NODE_COMPONENT',
    IMAGE_COMPONENT: 'IMAGE_COMPONENT',
    CVE: 'CVE',
    IMAGE_CVE: 'IMAGE_CVE',
    NODE_CVE: 'NODE_CVE',
    CLUSTER_CVE: 'CLUSTER_CVE',
    POLICY: 'POLICY',
    CONTROL: 'CONTROL',
};

export type RbacConfigType = 'SUBJECT' | 'SERVICE_ACCOUNT' | 'ROLE';

export const rbacConfigTypes: Record<RbacConfigType, RbacConfigType> = {
    SUBJECT: 'SUBJECT',
    SERVICE_ACCOUNT: 'SERVICE_ACCOUNT',
    ROLE: 'ROLE',
};

export type AccessControlEntityType = 'ACCESS_SCOPE' | 'AUTH_PROVIDER' | 'PERMISSION_SET' | 'ROLE';

export type StandardEntityType = 'CONTROL' | 'CATEGORY' | 'STANDARD' | 'CHECK';

export const standardEntityTypes: Record<StandardEntityType, StandardEntityType> = {
    CONTROL: 'CONTROL',
    CATEGORY: 'CATEGORY',
    STANDARD: 'STANDARD',
    CHECK: 'CHECK',
};

export const standardTypes = {
    PCI_DSS_3_2: 'PCI_DSS_3_2',
    NIST_800_190: 'NIST_800_190',
    NIST_SP_800_53_Rev_4: 'NIST_SP_800_53_Rev_4',
    HIPAA_164: 'HIPAA_164',
    CIS_Kubernetes_v1_5: 'CIS_Kubernetes_v1_5',
    CIS_Docker_v1_1_0: 'CIS_Docker_v1_1_0',
    CIS_Docker_v1_2_0: 'CIS_Docker_v1_2_0',
};

export const standardBaseTypes = {
    [standardTypes.PCI_DSS_3_2]: 'PCI',
    [standardTypes.NIST_800_190]: 'NIST SP 800-190',
    [standardTypes.NIST_SP_800_53_Rev_4]: 'NIST SP 800-53',
    [standardTypes.HIPAA_164]: 'HIPAA',
    [standardTypes.CIS_Kubernetes_v1_5]: 'CIS K8s',
};

export const searchCategories: Record<string, SearchCategory> = {
    NAMESPACE: 'NAMESPACES',
    NODE: 'NODES',
    CLUSTER: 'CLUSTERS',
    CONTROL: 'COMPLIANCE',
    CVE: 'VULNERABILITIES',
    CLUSTER_CVE: 'CLUSTER_VULNERABILITIES',
    IMAGE_CVE: 'IMAGE_VULNERABILITIES',
    NODE_CVE: 'NODE_VULNERABILITIES',
    COMPONENT: 'IMAGE_COMPONENTS',
    IMAGE_COMPONENT: 'IMAGE_COMPONENTS',
    NODE_COMPONENT: 'NODE_COMPONENTS',
    DEPLOYMENT: 'DEPLOYMENTS',
    SECRET: 'SECRETS',
    POLICY: 'POLICIES',
    IMAGE: 'IMAGES',
    REPORT_CONFIGURATIONS: 'REPORT_CONFIGURATIONS',
    RISK: 'RISKS',
    ROLE: 'ROLES',
    SERVICE_ACCOUNT: 'SERVICE_ACCOUNTS',
    SUBJECT: 'SUBJECTS',
};

export default {
    ...resourceTypes,
    ...standardTypes,
    ...standardEntityTypes,
    ...rbacConfigTypes,
} as const;
