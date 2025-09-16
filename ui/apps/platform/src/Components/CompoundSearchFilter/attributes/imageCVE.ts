// If you're adding a new attribute, make sure to add it to "imageCVEAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';
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

/*
// Ross CISA KEV
export const KnownExploit: CompoundSearchFilterAttribute = {
    displayName: 'Known exploit',
    filterChipLabel: 'Known exploit',
    searchTerm: 'Known Exploit',
    inputType: 'select',
    inputProps: {
        hasCheckbox: false,
        options: [
            { label: 'Has a known expoit', value: 'HAS_KNOWN_EXPLOIT' },
            // {
            //     label: 'Used in ransomware campaigns',
            //     value: 'USED_IN_RANSOMWARE',
            // },
            { label: 'No known exploit', value: 'NO_KNOWN_EXPLOIT' },
        ],
    },
    // featureFlagDependency: ['ROX_SCANNER_V4', 'ROX_KEV_EXPLOIT'],
    featureFlagDependency: ['ROX_SCANNER_V4'],
};

export const imageCVEAttributes = [Name, DiscoveredTime, CVSS, EPSSProbability, KnownExploit];
*/
export const imageCVEAttributes = [Name, DiscoveredTime, CVSS, EPSSProbability];
