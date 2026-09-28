// If you're adding a new attribute, make sure to add it to "imageCVEAttributes" as well

import { originDisplayNames } from 'Containers/Vulnerabilities/utils/vulnerabilityUtils';

import type { CompoundSearchFilterAttribute } from '../types';
import { EPSSProbability } from './epssProbability';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Image CVE',
    searchTerm: 'CVE',
    inputType: 'autocomplete',
};

export const DiscoveredTime: CompoundSearchFilterAttribute = {
    displayName: 'Discovered time',
    filterChipLabel: 'Image CVE discovered time',
    searchTerm: 'CVE Created Time',
    inputType: 'date-picker',
};

export const CVSS: CompoundSearchFilterAttribute = {
    displayName: 'CVSS',
    filterChipLabel: 'CVSS',
    searchTerm: 'CVSS',
    inputType: 'condition-number',
};

export const KnownExploit: CompoundSearchFilterAttribute = {
    displayName: 'Known exploit',
    filterChipLabel: 'Known exploit',
    searchTerm: 'CISA KEV', // and 'Known Ransomware Campaign' as category2
    inputType: 'select-exclusive-double',
    inputProps: {
        category2: 'Known Ransomware Campaign',
        options: [
            {
                label: 'Has a known exploit',
                category: 'CISA KEV',
                value: 'true',
            },
            {
                label: 'Used in ransomware campaigns',
                category: 'Known Ransomware Campaign',
                value: 'true',
            },
            {
                label: 'No known exploit',
                category: 'CISA KEV',
                value: 'false',
            },
        ],
    },
    featureFlagDependency: ['ROX_SCANNER_V4', 'ROX_CISA_KEV'],
};

// The filter value is the VulnOrigin enum name (e.g. VULN_ORIGIN_RED_HAT), which the
// backend enum search matches by case-insensitive prefix. The human-readable label
// would not match, so value must be the enum key, not the display name.
//
// Intentionally NOT part of imageCVEAttributes: the CVE origin filter is only shown on
// the single image page, single deployment page, and image vulnerability reports, which
// append it to their own configs.
export const Origin: CompoundSearchFilterAttribute = {
    displayName: 'Origin',
    filterChipLabel: 'CVE origin',
    searchTerm: 'CVE Origin',
    inputType: 'select',
    inputProps: {
        options: Object.entries(originDisplayNames).map(([value, label]) => ({
            value,
            label: label ?? value,
        })),
    },
    featureFlagDependency: ['ROX_SCANNER_V4'],
};

export const imageCVEAttributes = [CVSS, DiscoveredTime, EPSSProbability, KnownExploit, Name];
