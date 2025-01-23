// If you're adding a new attribute, make sure to add it to "imageCVEAttributes" as well

import { CompoundSearchFilterAttribute, ConditionTextFilterAttribute } from '../types';

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

export const EPSSProbability: ConditionTextFilterAttribute = {
    displayName: 'EPSS probability',
    filterChipLabel: 'EPSS probability',
    searchTerm: 'EPSS Probability',
    inputType: 'condition-text',
    // featureFlagDependency: ['ROX_SCANNER_V4', 'ROX_EPSS_SCORE'],
    featureFlagDependency: ['ROX_SCANNER_V4'],
    inputProps: {
        conditionProps: {
            conditionEntries: [
                ['>', 'Is greater than'],
                ['>=', 'Is greater than or equal to'],
                ['=', 'Is equal to'],
                ['<=', 'Is less than or equal to'],
                ['<', 'Is less than'],
            ],
        },
        textProps: {
            convertFromExternalToInternalText: (externalText: string) => {
                const float = parseFloat(externalText); // assume validateExternalText
                return String(float / 100); // from percent to fraction
            },
            convertFromInternalToExternalText: (internalText: string) => {
                const float = parseFloat(internalText); // assume validateInternalText
                const percent = float * 100; // from fraction to percent
                return `${percent.toFixed(3)}%`; // same precison as EPSS data
            },
            externalTextDefault: '0%',
            validateExternalText: (externalText: string) => {
                if (!/^(\.\d+|\d+(?:\.\d*)?)%?$/.test(externalText.trim())) {
                    return false;
                }
                const float = parseFloat(externalText);
                return !Number.isNaN(float) && float >= 0 && float <= 100;
            },
            validateInternalText: (internalText: string) => {
                // Assume internalText was serialized by String (see above).
                // Leading zero followed by optional decimal point and digits.
                if (!/^(\.\d+|\d+(?:\.\d*)?)$/.test(internalText.trim())) {
                    return false;
                }
                const float = parseFloat(internalText);
                return !Number.isNaN(float) && float >= 0 && float <= 1;
            },
        },
    },
} as const;

export const imageCVEAttributes = [Name, DiscoveredTime, CVSS, EPSSProbability];
