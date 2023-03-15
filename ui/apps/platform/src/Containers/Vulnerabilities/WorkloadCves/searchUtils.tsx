import qs from 'qs';

import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';
import { ensureExhaustive } from 'utils/type.utils';

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

export function getOverviewCvesPath(workloadCvesSearch: WorkloadCvesSearch): string {
    return `${vulnerabilitiesWorkloadCvesPath}${getQueryString(workloadCvesSearch)}`;
}

export function getEntityPagePath(workloadCveEntity: EntityTab, id: string): string {
    switch (workloadCveEntity) {
        case 'CVE':
            return `${vulnerabilitiesWorkloadCvesPath}/cves/${id}`;
        case 'Image':
            return `${vulnerabilitiesWorkloadCvesPath}/images/${id}`;
        case 'Deployment':
            return `${vulnerabilitiesWorkloadCvesPath}/deployments/${id}`;
        default:
            return ensureExhaustive(workloadCveEntity);
    }
}
