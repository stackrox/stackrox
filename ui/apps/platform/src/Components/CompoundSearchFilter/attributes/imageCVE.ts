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

export const imageCVEAttributes = [Name, DiscoveredTime, CVSS, EPSSProbability];
