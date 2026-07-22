import { sourceTypeLabels, sourceTypes } from 'Containers/Vulnerabilities/constants';
import type { CompoundSearchFilterAttribute, SelectSearchFilterAttribute } from '../types';

export const VirtualMachineCVEName: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'CVE',
    searchTerm: 'CVE',
    inputType: 'text',
};

export const VirtualMachineComponentName: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Virtual machine component name',
    searchTerm: 'Component',
    inputType: 'text',
};

export const VirtualMachineComponentVersion: CompoundSearchFilterAttribute = {
    displayName: 'Version',
    filterChipLabel: 'Virtual machine component version',
    searchTerm: 'Component Version',
    inputType: 'text',
};

export const VirtualMachineComponentSource: SelectSearchFilterAttribute = {
    displayName: 'Source',
    filterChipLabel: 'Virtual machine component source',
    searchTerm: 'Component Source',
    inputType: 'select',
    inputProps: {
        options: sourceTypes.map((sourceType) => ({
            label: sourceTypeLabels[sourceType],
            value: sourceType,
        })),
    },
};

export const VirtualMachineID: CompoundSearchFilterAttribute = {
    displayName: 'ID',
    filterChipLabel: 'Virtual machine ID',
    searchTerm: 'Virtual Machine ID',
    inputType: 'text',
};

export const VirtualMachineName: CompoundSearchFilterAttribute = {
    displayName: 'Name',
    filterChipLabel: 'Virtual machine name',
    searchTerm: 'Virtual Machine Name',
    inputType: 'text',
};
