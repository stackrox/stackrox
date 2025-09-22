import { CompoundSearchFilterAttribute } from '../types';

export const VirtualMachineCVEName: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'CVE',
    searchTerm: 'CVE',
    inputType: 'text',
};

export const VirtualMachineCVSS: CompoundSearchFilterAttribute = {
    displayName: 'CVSS',
    filterChipLabel: 'CVE CVSS',
    searchTerm: 'CVSS',
    inputType: 'condition-number',
};

export const VirtualMachineComponentName: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Component',
    searchTerm: 'Component',
    inputType: 'text',
};

export const VirtualMachineComponentCVSS: CompoundSearchFilterAttribute = {
    displayName: 'Version',
    filterChipLabel: 'Component version',
    searchTerm: 'Component Version',
    inputType: 'text',
};
