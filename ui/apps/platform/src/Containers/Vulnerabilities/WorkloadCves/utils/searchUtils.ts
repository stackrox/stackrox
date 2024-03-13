import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import { VulnerabilityState } from 'types/cve.proto';
import { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';

import { WorkloadEntityTab } from '../../types';

type WorkloadCvesSearch = {
    vulnerabilityState: VulnerabilityState;
    entityTab?: WorkloadEntityTab;
    s?: SearchFilter;
};

export function getOverviewCvesPath(workloadCvesSearch: WorkloadCvesSearch): string {
    return `${vulnerabilitiesWorkloadCvesPath}${getQueryString(workloadCvesSearch)}`;
}
