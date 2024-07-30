// If you're adding a new attribute, make sure to add it to the "platformCVEAttributes" array as well

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

const Type = {
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

export const platformCVEAttributes = [Name, DiscoveredTime, CVSS, Type] as const;

export type PlatformCVEAttribute = (typeof platformCVEAttributes)[number]['displayName'];

export function getPlatformCVEAttributes(
    attributes?: PlatformCVEAttribute[]
): SearchFilterAttribute[] {
    if (!attributes || attributes.length === 0) {
        return platformCVEAttributes as unknown as SearchFilterAttribute[];
    }

    return platformCVEAttributes.filter((imageAttribute) => {
        return attributes.includes(imageAttribute.displayName);
    }) as SearchFilterAttribute[];
}
