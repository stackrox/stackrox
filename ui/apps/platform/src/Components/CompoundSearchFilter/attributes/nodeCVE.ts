// If you're adding a new attribute, make sure to add it to the "nodeCVEAttributes" object as well

export const Name = {
    displayName: 'Name',
    filterChipLabel: 'Node CVE',
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

export const nodeCVEAttributes = { Name, DiscoveredTime, CVSS } as const;
