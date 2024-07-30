// If you're adding a new attribute, make sure to add it to the "imageCVEAttributes" array as well

import { SearchFilterAttribute } from '../types';

const Name = {
    displayName: 'Name',
    filterChipLabel: 'Image CVE',
    searchTerm: 'CVE',
    inputType: 'autocomplete',
} as const;

const DiscoveredTime = {
    displayName: 'Discovered time',
    filterChipLabel: 'Image CVE discovered time',
    searchTerm: 'CVE Created Time',
    inputType: 'date-picker',
} as const;

const CVSS = {
    displayName: 'CVSS',
    filterChipLabel: 'CVSS',
    searchTerm: 'CVSS',
    inputType: 'condition-number',
} as const;

export const imageCVEAttributes = [Name, DiscoveredTime, CVSS] as const;

export type ImageCVEAttribute = (typeof imageCVEAttributes)[number]['displayName'];

export function getImageCVEAttributes(attributes?: ImageCVEAttribute[]): SearchFilterAttribute[] {
    if (!attributes || attributes.length === 0) {
        return imageCVEAttributes as unknown as SearchFilterAttribute[];
    }

    return imageCVEAttributes.filter((imageAttribute) => {
        return attributes.includes(imageAttribute.displayName);
    }) as SearchFilterAttribute[];
}
