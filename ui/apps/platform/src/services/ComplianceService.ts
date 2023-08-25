import axios from './instance';
import { Empty } from './types';

const standardsUrl = '/v1/compliance/standards';

export type ComplianceStandardScope = 'UNSET' | 'CLUSTER' | 'NAMESPACE' | 'DEPLOYMENT' | 'NODE';

export type ComplianceStandardMetadata = {
    id: string;
    name: string;
    description: string;
    numImplementedChecks: number; // int32
    scopes: ComplianceStandardScope[];
    dynamic: boolean;
    hideScanResults: boolean;
};

export function fetchComplianceStandards(): Promise<ComplianceStandardMetadata[]> {
    return axios
        .get<{ standards: ComplianceStandardMetadata[] }>(standardsUrl)
        .then((response) => response.data?.standards ?? []);
}

/*
 * Temporary frontend work-around for 4.1 version,
 * because standards are not always sorted in ascending order by name.
 */

function compareStandardsByName(
    standardPrev: ComplianceStandardMetadata,
    standardNext: ComplianceStandardMetadata
) {
    const { name: namePrev } = standardPrev;
    const { name: nameNext } = standardNext;

    /* eslint-disable no-nested-ternary */
    return namePrev < nameNext ? -1 : namePrev > nameNext ? 1 : 0;
    /* eslint-enable no-nested-ternary */
}

export function fetchComplianceStandardsSortedByName(): Promise<ComplianceStandardMetadata[]> {
    return fetchComplianceStandards().then((standards) => standards.sort(compareStandardsByName));
}

export function patchComplianceStandard(id: string, hideScanResults: boolean): Promise<Empty> {
    return axios
        .patch<Empty>(`${standardsUrl}/${id}`, { hideScanResults })
        .then((response) => response.data);
}
