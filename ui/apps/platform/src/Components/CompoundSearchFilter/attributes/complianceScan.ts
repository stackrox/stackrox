// If you're adding a new attribute, make sure to add it to "complianceScanAttributes" as well

import { CompoundSearchFilterAttribute } from '../types';

export const ConfigID: CompoundSearchFilterAttribute = {
    displayName: 'Config ID',
    filterChipLabel: 'Compliance scan config ID',
    searchTerm: 'Compliance Scan Config Id',
    inputType: 'text',
};

export const complianceScanAttributes = [ConfigID];
