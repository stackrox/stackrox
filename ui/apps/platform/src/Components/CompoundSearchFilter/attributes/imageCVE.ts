// If you're adding a new attribute, make sure to add it to the "imageCVEAttributes" object as well

export const Name = {
    displayName: 'Name',
    filterChipLabel: 'Image CVE',
    searchTerm: 'CVE',
    inputType: 'autocomplete',
} as const;

export const DiscoveredTime = {
    displayName: 'Discovered time',
    filterChipLabel: 'Image CVE discovered time',
    searchTerm: 'CVE Created Time',
    inputType: 'date-picker',
} as const;

export const CVSS = {
    displayName: 'CVSS',
    filterChipLabel: 'CVSS',
    searchTerm: 'CVSS',
    inputType: 'condition-number',
} as const;

export const imageCVEAttributes = { Name, DiscoveredTime, CVSS } as const;
