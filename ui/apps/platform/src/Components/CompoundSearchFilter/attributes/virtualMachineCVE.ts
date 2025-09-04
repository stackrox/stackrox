// If you're adding a new attribute, make sure to add it to "virtualMachineCVEAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';
import { EPSSProbability } from './epssProbability';

export const Name: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Virtual Machine CVE',
    searchTerm: 'CVE',
    inputType: 'autocomplete',
};

export const DiscoveredTime: CompoundSearchFilterAttribute = {
    displayName: 'Discovered time',
    filterChipLabel: 'Virtual Machine CVE discovered time',
    searchTerm: 'CVE Created Time',
    inputType: 'date-picker',
};

export const CVSS: CompoundSearchFilterAttribute = {
    displayName: 'CVSS',
    filterChipLabel: 'CVSS',
    searchTerm: 'CVSS',
    inputType: 'condition-number',
};

export const virtualMachineCVEAttributes = [Name, DiscoveredTime, CVSS, EPSSProbability];
