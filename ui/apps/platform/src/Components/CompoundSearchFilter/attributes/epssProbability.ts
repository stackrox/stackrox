import { ConditionTextFilterAttribute } from '../types';
import { ConditionEntries } from '../components/ConditionText';

const conditionEntries: ConditionEntries = [
    ['>', 'Is greater than'],
    ['>=', 'Is greater than or equal to'],
    // Intentionally omit = because potential problem with floating point
    ['<=', 'Is less than or equal to'],
    ['<', 'Is less than'],
];

export function convertFromExternalToInternalText(externalText: string) {
    const float = parseFloat(externalText); // assume validateExternalText
    return String(float / 100); // from percent to fraction
}

export function convertFromInternalToExternalText(internalText: string) {
    const float = parseFloat(internalText); // assume validateInternalText
    const percent = float * 100; // from fraction to percent
    return `${percent.toFixed(3)}%`; // same precison as EPSS data
}

export const externalTextDefault = '0%';

export const externalTextRegExp = /^(\.\d+|\d+(?:\.\d*)?)%?$/;

export function validateExternalText(externalText: string) {
    if (!externalTextRegExp.test(externalText.trim())) {
        return false;
    }
    const float = parseFloat(externalText);
    return !Number.isNaN(float) && float >= 0 && float <= 100;
}

export const internalTextRegExp = /^(\.\d+|\d+(?:\.\d*)?)$/;

export function validateInternalText(internalText: string) {
    if (!internalTextRegExp.test(internalText.trim())) {
        return false;
    }
    const float = parseFloat(internalText);
    return !Number.isNaN(float) && float >= 0 && float <= 1;
}

export const EPSSProbability: ConditionTextFilterAttribute = {
    displayName: 'EPSS probability',
    filterChipLabel: 'EPSS probability',
    searchTerm: 'EPSS Probability',
    inputType: 'condition-text',
    featureFlagDependency: ['ROX_SCANNER_V4'],
    inputProps: {
        conditionProps: {
            conditionEntries,
        },
        textProps: {
            convertFromExternalToInternalText,
            convertFromInternalToExternalText,
            externalTextDefault,
            validateExternalText,
            validateInternalText,
        },
    },
} as const;
