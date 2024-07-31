import { SearchCategory } from 'services/SearchService';
import { getClusterAttributes } from 'Components/CompoundSearchFilter/attributes/cluster';
import { getProfileCheckAttributes } from 'Components/CompoundSearchFilter/attributes/profileCheck';

const profileCheckSearchFilterConfig = {
    displayName: 'Profile check',
    searchCategory: 'COMPLIANCE' as SearchCategory,
    attributes: getProfileCheckAttributes(),
};

const clusterSearchFilterConfig = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS' as SearchCategory,
    attributes: getClusterAttributes(),
};

export { profileCheckSearchFilterConfig, clusterSearchFilterConfig };
