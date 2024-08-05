// If you're adding a new attribute, make sure to add it to "platformCVEAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Platform CVE',
    searchTerm: 'CVE',
    inputType: 'autocomplete',
};

export const DiscoveredTime: CompoundSearchFilterAttribute = {
    displayName: 'Discovered time',
    filterChipLabel: 'CVE discovered time',
    searchTerm: 'CVE Created Time',
    inputType: 'date-picker',
};

export const CVSS: CompoundSearchFilterAttribute = {
    displayName: 'CVSS',
    filterChipLabel: 'CVE CVSS',
    searchTerm: 'CVSS',
    inputType: 'condition-number',
};

export const Type: CompoundSearchFilterAttribute = {
    displayName: 'Type',
    filterChipLabel: 'CVE type',
    searchTerm: 'CVE Type',
    inputType: 'select',
    inputProps: {
        options: [
            { label: 'K8s CVE', value: 'K8S_CVE' },
            { label: 'Istio CVE', value: 'ISTIO_CVE' },
            { label: 'Openshift CVE', value: 'OPENSHIFT_CVE' },
        ],
    },
};

export const platformCVEAttributes = [Name, DiscoveredTime, CVSS, Type];
