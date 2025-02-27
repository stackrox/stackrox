import { clusterAttributes } from 'Components/CompoundSearchFilter/attributes/cluster';
import { profileCheckAttributes } from 'Components/CompoundSearchFilter/attributes/profileCheck';
import { CompoundSearchFilterEntity } from 'Components/CompoundSearchFilter/types';

export const profileCheckSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Profile check',
    searchCategory: 'COMPLIANCE',
    attributes: profileCheckAttributes,
};

export const clusterSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: clusterAttributes,
};
