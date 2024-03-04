import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import { VulnerabilityState } from 'types/cve.proto';
import { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';

import { EntityTab } from '../../types';

type WorkloadCvesSearch = {
    vulnerabilityState: VulnerabilityState;
    entityTab?: EntityTab;
    s?: SearchFilter;
};

export function getOverviewCvesPath(workloadCvesSearch: WorkloadCvesSearch): string {
    return `${vulnerabilitiesWorkloadCvesPath}${getQueryString(workloadCvesSearch)}`;
}
