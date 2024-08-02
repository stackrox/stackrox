import { SearchCategory } from 'services/SearchService';
import { clusterAttributes } from 'Components/CompoundSearchFilter/attributes/cluster';
import { profileCheckAttributes } from 'Components/CompoundSearchFilter/attributes/profileCheck';

const profileCheckSearchFilterConfig = {
    displayName: 'Profile check',
    searchCategory: 'COMPLIANCE' as SearchCategory,
    attributes: profileCheckAttributes,
};

const clusterSearchFilterConfig = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS' as SearchCategory,
    attributes: clusterAttributes,
};

export { profileCheckSearchFilterConfig, clusterSearchFilterConfig };
