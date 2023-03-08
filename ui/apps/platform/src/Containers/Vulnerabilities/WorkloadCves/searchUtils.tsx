import qs from 'qs';
import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';

import { CveStatusTab, isValidCveStatusTab } from './types';

export type EntityTab = 'CVE' | 'Image' | 'Deployment';

export type WorkloadCvesSearch = {
    cveStatusTab: CveStatusTab;
    entityTab?: EntityTab;
    s?: SearchFilter;
};

export function parseWorkloadCvesOverviewSearchString(search: string): WorkloadCvesSearch {
    const { cveStatusTab } = qs.parse(search, { ignoreQueryPrefix: true });

    return {
        cveStatusTab: isValidCveStatusTab(cveStatusTab) ? cveStatusTab : 'Observed',
    };
}

export function getOverviewCvesPath(workloadCvesSearch: WorkloadCvesSearch) {
    return `${vulnerabilitiesWorkloadCvesPath}${getQueryString(workloadCvesSearch)}`;
}
