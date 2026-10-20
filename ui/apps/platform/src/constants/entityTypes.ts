import type { SearchCategory } from 'services/SearchService';

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
    | 'POLICY';

export const resourceTypes = {
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
} as const;

export type RbacConfigType = 'SUBJECT' | 'SERVICE_ACCOUNT' | 'ROLE';

export const rbacConfigTypes: Record<RbacConfigType, RbacConfigType> = {
    SUBJECT: 'SUBJECT',
    SERVICE_ACCOUNT: 'SERVICE_ACCOUNT',
    ROLE: 'ROLE',
};

export type AccessControlEntityType = 'ACCESS_SCOPE' | 'AUTH_PROVIDER' | 'PERMISSION_SET' | 'ROLE';

export const searchCategories: Record<string, SearchCategory> = {
    NAMESPACE: 'NAMESPACES',
    NODE: 'NODES',
    CLUSTER: 'CLUSTERS',
    CVE: 'VULNERABILITIES',
    CLUSTER_CVE: 'CLUSTER_VULNERABILITIES',
    IMAGE_CVE: 'IMAGE_VULNERABILITIES_V2', // flat CVE data model
    NODE_CVE: 'NODE_VULNERABILITIES',
    IMAGE_COMPONENT: 'IMAGE_COMPONENTS_V2', // flat CVE data model
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
    ...rbacConfigTypes,
} as const;
