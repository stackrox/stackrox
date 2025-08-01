import {
    clusterIdAttribute,
    clusterLabelAttribute,
    clusterNameAttribute,
} from 'Components/CompoundSearchFilter/attributes/cluster';
import { profileCheckAttributes } from 'Components/CompoundSearchFilter/attributes/profileCheck';
import type { CompoundSearchFilterEntity } from 'Components/CompoundSearchFilter/types';
import type { FilterChipGroupDescriptor } from 'Components/PatternFly/SearchFilterChips';

import { CHECK_STATUS_QUERY } from './Coverage/compliance.coverage.constants';

export const profileCheckSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Profile check',
    searchCategory: 'COMPLIANCE',
    attributes: profileCheckAttributes,
};

export const clusterSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: [clusterIdAttribute, clusterNameAttribute, clusterLabelAttribute],
};

export const complianceStatusFilterChipDescriptors: FilterChipGroupDescriptor = {
    displayName: 'Compliance status',
    searchFilterName: CHECK_STATUS_QUERY,
};
