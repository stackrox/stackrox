import axios from 'services/instance';
import qs from 'qs';

import type { SearchFilter } from 'types/search';
import { applyRegexSearchModifiers, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import { complianceV2Url } from './ComplianceCommon';
import type { ComplianceProfileSummary } from './ComplianceCommon';

const complianceProfilesBaseUrl = `${complianceV2Url}/profiles`;

/**
 * Fetches a list of compliance profile summaries based on the provided cluster IDs and search filter.
 */
export function listProfileSummaries(
    clusterIds: string[],
    searchFilter?: SearchFilter
): Promise<ComplianceProfileSummary[]> {
    const query = searchFilter
        ? getRequestQueryStringForSearchFilter(applyRegexSearchModifiers(searchFilter))
        : '';
    const queryParams: Record<string, unknown> = { cluster_ids: clusterIds };
    if (query) {
        queryParams.query = { query };
    }
    const params = qs.stringify(queryParams, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<{
            profiles: ComplianceProfileSummary[];
        }>(`${complianceProfilesBaseUrl}/summary?${params}`)
        .then((response) => {
            return response?.data?.profiles ?? [];
        });
}
