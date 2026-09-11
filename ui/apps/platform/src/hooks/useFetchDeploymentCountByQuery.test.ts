import { renderHook } from '@testing-library/react';

import { fetchDeploymentsCountByQuery } from 'services/DeploymentsService';
import waitForNextUpdate from 'test-utils/waitForNextUpdate';

import useFetchDeploymentCountByQuery from './useFetchDeploymentCountByQuery';

vi.mock('services/DeploymentsService', () => ({
    fetchDeploymentsCountByQuery: vi.fn(),
}));

const mockedFetchDeploymentsCountByQuery = vi.mocked(fetchDeploymentsCountByQuery);

const scopedQuery = 'Cluster:production+Vulnerability State:OBSERVED';

describe('useFetchDeploymentCountByQuery', () => {
    beforeEach(() => {
        mockedFetchDeploymentsCountByQuery.mockReset();
    });

    it('returns the REST deployment count for the given query', async () => {
        vi.useFakeTimers({ shouldAdvanceTime: true });
        mockedFetchDeploymentsCountByQuery.mockResolvedValue(4);

        const { result } = renderHook(() => useFetchDeploymentCountByQuery(scopedQuery));

        await waitForNextUpdate(result);

        expect(result.current.isLoading).toBe(false);
        expect(result.current.data).toBe(4);
        expect(result.current.error).toBeUndefined();
        expect(mockedFetchDeploymentsCountByQuery).toHaveBeenCalledWith(scopedQuery);
    });
});
