import qs from 'qs';

import { SearchFilter, ApiSortOption } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { mockComplianceScanResultsOverview } from 'Containers/ComplianceEnhanced/Status/MockData/complianceScanResultsOverview';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';

interface ComplianceScanStatsShim {
    scanName: string;
    numberOfChecks: number; // int32
    numberOfFailingChecks: number; // int32
    numberOfPassingChecks: number; // int32
    lastScan: string; // ISO 8601 date string
}

export interface ComplianceScanResultsOverview {
    scanStats: ComplianceScanStatsShim;
    profileName: string[];
    clusterId: string[];
}

export interface ListComplianceScanResultsOverviewResponse {
    scanOverviews: ComplianceScanResultsOverview[];
}

export function complianceResultsOverview(
    searchFilter: SearchFilter,
    sortOption: ApiSortOption,
    page?: number,
    pageSize?: number
): CancellableRequest<ComplianceScanResultsOverview[]> {
    let offset: number | undefined;
    if (typeof page === 'number' && typeof pageSize === 'number') {
        offset = page > 0 ? page * pageSize : 0;
    }
    const query = {
        query: getRequestQueryStringForSearchFilter(searchFilter),
        pagination: { offset, limit: pageSize, sortOption },
    };
    // TODO: remove disabled linter rule when service updated
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const params = qs.stringify({ query }, { allowDots: true });
    return makeCancellableAxiosRequest((signal) => {
        return new Promise((resolve, reject) => {
            if (!signal.aborted) {
                setTimeout(() => {
                    const mockData = mockComplianceScanResultsOverview();
                    resolve(mockData.scanOverviews);
                }, 2000);
            } else {
                reject(new Error('Request was aborted'));
            }
        });
    });
}
