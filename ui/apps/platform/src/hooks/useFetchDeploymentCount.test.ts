import { renderHook } from '@testing-library/react';

import { fetchDeploymentsCount } from 'services/DeploymentsService';
import waitForNextUpdate from 'test-utils/waitForNextUpdate';

import useFetchDeploymentCount from './useFetchDeploymentCount';

vi.mock('services/DeploymentsService', () => ({
    fetchDeploymentsCount: vi.fn(),
}));

const mockedFetchDeploymentsCount = vi.mocked(fetchDeploymentsCount);

const searchFilter = { Cluster: 'production', Namespace: 'stackrox' };

describe('useFetchDeploymentCount', () => {
    beforeEach(() => {
        mockedFetchDeploymentsCount.mockReset();
    });

    it('returns the REST deployment count for the given search filter', async () => {
        vi.useFakeTimers({ shouldAdvanceTime: true });
        mockedFetchDeploymentsCount.mockResolvedValue(12);

        const { result } = renderHook(() => useFetchDeploymentCount(searchFilter));

        expect(result.current.isLoading).toBe(true);
        expect(result.current.data).toBeUndefined();
        expect(result.current.error).toBeUndefined();

        await waitForNextUpdate(result);

        expect(result.current.isLoading).toBe(false);
        expect(result.current.data).toBe(12);
        expect(result.current.error).toBeUndefined();
        expect(mockedFetchDeploymentsCount).toHaveBeenCalledWith(searchFilter);
    });

    it('returns a zero count', async () => {
        vi.useFakeTimers({ shouldAdvanceTime: true });
        mockedFetchDeploymentsCount.mockResolvedValue(0);

        const { result } = renderHook(() => useFetchDeploymentCount(searchFilter));

        await waitForNextUpdate(result);

        expect(result.current.isLoading).toBe(false);
        expect(result.current.data).toBe(0);
        expect(result.current.error).toBeUndefined();
    });

    it('returns the REST error', async () => {
        vi.useFakeTimers({ shouldAdvanceTime: true });
        const requestError = new Error('count failed');
        mockedFetchDeploymentsCount.mockRejectedValue(requestError);

        const { result } = renderHook(() => useFetchDeploymentCount(searchFilter));

        await waitForNextUpdate(result);

        expect(result.current.isLoading).toBe(false);
        expect(result.current.data).toBeUndefined();
        expect(result.current.error).toBe(requestError);
    });
});
