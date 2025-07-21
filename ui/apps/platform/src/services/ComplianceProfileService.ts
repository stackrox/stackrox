import axios from 'services/instance';
import qs from 'qs';

import { complianceV2Url } from './ComplianceCommon';
import type { ComplianceProfileSummary } from './ComplianceCommon';

const complianceProfilesBaseUrl = `${complianceV2Url}/profiles`;

/**
 * Fetches a list of compliance profile summaries based on the provided cluster IDs.
 */
export function listProfileSummaries(clusterIds): Promise<ComplianceProfileSummary[]> {
    const params = qs.stringify({ cluster_ids: clusterIds }, { arrayFormat: 'repeat' });
    return axios
        .get<{
            profiles: ComplianceProfileSummary[];
        }>(`${complianceProfilesBaseUrl}/summary?${params}`)
        .then((response) => {
            return response?.data?.profiles ?? [];
        });
}
