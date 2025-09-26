import { CompoundSearchFilterAttribute } from '../types';

export const VirtualMachineCVEName: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'CVE',
    searchTerm: 'CVE',
    inputType: 'text',
};

export const VirtualMachineComponentName: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Component name',
    searchTerm: 'Component',
    inputType: 'text',
};

export const VirtualMachineComponentVersion: CompoundSearchFilterAttribute = {
    displayName: 'Version',
    filterChipLabel: 'Component version',
    searchTerm: 'Component Version',
    inputType: 'text',
};
