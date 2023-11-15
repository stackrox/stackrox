import { SearchCategory } from 'services/SearchService';

export type SearchOptionValue =
    | 'SEVERITY'
    | 'FIXABLE'
    | 'CVE'
    | 'IMAGE'
    | 'DEPLOYMENT'
    | 'NAMESPACE'
    | 'CLUSTER'
    | 'COMPONENT'
    | 'COMPONENT SOURCE'
    | 'Request Name'
    | 'Requester User Name';

export type SearchOption = { label: string; value: SearchOptionValue; category: SearchCategory };

export const SEVERITY_SEARCH_OPTION = {
    label: 'Severity',
    value: 'SEVERITY',
    category: 'IMAGE_VULNERABILITIES',
} as const;

export const FIXABLE_SEARCH_OPTION = {
    label: 'Fixable',
    value: 'FIXABLE',
    category: 'IMAGE_VULNERABILITIES',
} as const;

export const IMAGE_CVE_SEARCH_OPTION = {
    label: 'CVE',
    value: 'CVE',
    category: 'IMAGE_VULNERABILITIES',
} as const;

export const IMAGE_SEARCH_OPTION = {
    label: 'Image',
    value: 'IMAGE',
    category: 'IMAGES',
} as const;

export const DEPLOYMENT_SEARCH_OPTION = {
    label: 'Deployment',
    value: 'DEPLOYMENT',
    category: 'DEPLOYMENTS',
} as const;

export const NAMESPACE_SEARCH_OPTION = {
    label: 'Namespace',
    value: 'NAMESPACE',
    category: 'NAMESPACES',
} as const;

export const CLUSTER_SEARCH_OPTION = {
    label: 'Cluster',
    value: 'CLUSTER',
    category: 'CLUSTERS',
} as const;

export const COMPONENT_SEARCH_OPTION = {
    label: 'Component',
    value: 'COMPONENT',
    category: 'IMAGE_COMPONENTS',
} as const;

export const COMPONENT_SOURCE_SEARCH_OPTION = {
    label: 'Component Source',
    value: 'COMPONENT SOURCE',
    category: 'IMAGE_VULNERABILITIES',
} as const;

export const REQUEST_NAME_SEARCH_OPTION = {
    label: 'Request name',
    value: 'Request Name',
    category: 'VULN_REQUEST', // This might need to change
} as const;

export const REQUESTER_SEARCH_OPTION = {
    label: 'Requester',
    value: 'Requester User Name',
    category: 'VULN_REQUEST', // This might need to change
} as const;
