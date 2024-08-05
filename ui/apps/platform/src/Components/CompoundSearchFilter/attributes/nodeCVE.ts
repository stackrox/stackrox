// If you're adding a new attribute, make sure to add it to "nodeCVEAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Node CVE',
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

export const nodeCVEAttributes = [Name, DiscoveredTime, CVSS];
