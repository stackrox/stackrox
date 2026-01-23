import type { GenericSearchFilterAttribute } from 'Components/CompoundSearchFilter/types';

import { EXTERNAL_SOURCE_ADDRESS_QUERY } from '../NetworkGraph.constants';

export const attributeForExternalSourceAddress: GenericSearchFilterAttribute = {
    displayName: 'Address', // theoretical because not in compound search filter
    filterChipLabel: 'IP or IP/CIDR',
    searchTerm: EXTERNAL_SOURCE_ADDRESS_QUERY,
    inputType: 'text',
};
