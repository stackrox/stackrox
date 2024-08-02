// If you're adding a new attribute, make sure to add it to the "platformCVEAttributes" object as well

export const Name = {
    displayName: 'Name',
    filterChipLabel: 'Platform CVE',
    searchTerm: 'CVE',
    inputType: 'autocomplete',
} as const;

export const DiscoveredTime = {
    displayName: 'Discovered time',
    filterChipLabel: 'CVE discovered time',
    searchTerm: 'CVE Created Time',
    inputType: 'date-picker',
} as const;

export const CVSS = {
    displayName: 'CVSS',
    filterChipLabel: 'CVE CVSS',
    searchTerm: 'CVSS',
    inputType: 'condition-number',
} as const;

export const Type = {
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
} as const;

export const platformCVEAttributes = { Name, DiscoveredTime, CVSS, Type } as const;
