// If you're adding a new attribute, make sure to add it to the "nodeCVEAttributes" array as well

import { SearchFilterAttribute } from '../types';

const Name = {
    displayName: 'Name',
    filterChipLabel: 'CVE',
    searchTerm: 'CVE',
    inputType: 'autocomplete',
} as const;

const DiscoveredTime = {
    displayName: 'Discovered time',
    filterChipLabel: 'CVE discovered time',
    searchTerm: 'CVE Created Time',
    inputType: 'date-picker',
} as const;

const CVSS = {
    displayName: 'CVSS',
    filterChipLabel: 'CVE CVSS',
    searchTerm: 'CVSS',
    inputType: 'condition-number',
} as const;

export const nodeCVEAttributes = [Name, DiscoveredTime, CVSS] as const;

export type NodeCVEAttribute = (typeof nodeCVEAttributes)[number]['displayName'];

export function getNodeCVEAttributes(attributes?: NodeCVEAttribute[]): SearchFilterAttribute[] {
    if (!attributes || attributes.length === 0) {
        return nodeCVEAttributes as unknown as SearchFilterAttribute[];
    }

    return nodeCVEAttributes.filter((imageAttribute) => {
        return attributes.includes(imageAttribute.displayName);
    }) as SearchFilterAttribute[];
}
