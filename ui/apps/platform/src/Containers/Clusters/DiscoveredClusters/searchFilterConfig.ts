import { statuses, types } from 'services/DiscoveredClusterService';
import type {
    CompoundSearchFilterConfig,
    CompoundSearchFilterEntity,
    SelectSearchFilterAttribute,
} from 'Components/CompoundSearchFilter/types';

import { getStatusText, getTypeText } from './DiscoveredCluster';

const attributeForStatus: SelectSearchFilterAttribute = {
    displayName: 'Status',
    filterChipLabel: 'Status',
    searchTerm: 'Cluster Status',
    inputType: 'select',
    inputProps: {
        options: statuses.map((value) => ({ label: getStatusText(value), value })),
    },
};

const attributeForType: SelectSearchFilterAttribute = {
    displayName: 'Type',
    filterChipLabel: 'Type',
    searchTerm: 'Cluster Type',
    inputType: 'select',
    inputProps: {
        options: types.map((value) => ({ label: getTypeText(value), value })),
    },
};

const entityForDiscoveredClusters: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: [attributeForStatus, attributeForType],
};

export const searchFilterConfig: CompoundSearchFilterConfig = [entityForDiscoveredClusters];
