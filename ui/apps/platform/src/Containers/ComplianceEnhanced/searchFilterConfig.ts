import {
    clusterIdAttribute,
    clusterLabelAttribute,
    clusterNameAttribute,
} from 'Components/CompoundSearchFilter/attributes/cluster';
import { profileCheckAttributes } from 'Components/CompoundSearchFilter/attributes/profileCheck';
import type {
    CompoundSearchFilterEntity,
    SelectSearchFilterAttribute,
} from 'Components/CompoundSearchFilter/types';

import { CHECK_STATUS_QUERY } from './Coverage/compliance.coverage.constants';

export const profileCheckSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Profile check',
    searchCategory: 'COMPLIANCE',
    attributes: profileCheckAttributes,
};

export const clusterSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: [clusterIdAttribute, clusterLabelAttribute, clusterNameAttribute],
};

export const attributeForComplianceCheckStatus: SelectSearchFilterAttribute = {
    displayName: 'Compliance status',
    filterChipLabel: 'Compliance status',
    searchTerm: CHECK_STATUS_QUERY,
    inputType: 'select',
    inputProps: {
        options: [
            { label: 'Pass', value: 'Pass' },
            { label: 'Fail', value: 'Fail' },
            { label: 'Error', value: 'Error' },
            { label: 'Info', value: 'Info' },
            { label: 'Manual', value: 'Manual' },
            { label: 'Not Applicable', value: 'Not Applicable' },
            { label: 'Inconsistent', value: 'Inconsistent' },
        ],
    },
};
