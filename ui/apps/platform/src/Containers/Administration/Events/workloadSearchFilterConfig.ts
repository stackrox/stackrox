import type { CompoundSearchFilterConfig } from 'Components/CompoundSearchFilter/types';
import {
    clusterSearchFilterConfig,
    deploymentSearchFilterConfig,
    namespaceSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';

export const workloadSearchFilterConfig: CompoundSearchFilterConfig = [
    clusterSearchFilterConfig,
    namespaceSearchFilterConfig,
    deploymentSearchFilterConfig,
];
