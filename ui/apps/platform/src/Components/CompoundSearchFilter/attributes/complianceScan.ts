// If you're adding a new attribute, make sure to add it to the "complianceScanAttributes" array as well

import { SearchFilterAttribute } from '../types';

const ConfigID = {
    displayName: 'Config ID',
    filterChipLabel: 'Compliance scan config ID',
    searchTerm: 'Compliance Scan Config Id',
    inputType: 'text',
} as const;

export const complianceScanAttributes = [ConfigID] as const;

export type ComplianceScanAttribute = (typeof complianceScanAttributes)[number]['displayName'];

export function getComplianceScanAttributes(
    attributes?: ComplianceScanAttribute[]
): SearchFilterAttribute[] {
    if (!attributes || attributes.length === 0) {
        return complianceScanAttributes as unknown as SearchFilterAttribute[];
    }

    return complianceScanAttributes.filter((imageAttribute) => {
        return attributes.includes(imageAttribute.displayName);
    }) as SearchFilterAttribute[];
}
