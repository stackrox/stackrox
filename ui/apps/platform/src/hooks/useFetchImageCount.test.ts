import { renderHook } from '@testing-library/react';

import { fetchImagesCount } from 'services/imageService';
import waitForNextUpdate from 'test-utils/waitForNextUpdate';

import useFetchImageCount from './useFetchImageCount';

vi.mock('services/imageService', () => ({
    fetchImagesCount: vi.fn(),
}));

const mockedFetchImagesCount = vi.mocked(fetchImagesCount);

const scopedQuery = 'Cluster:production+Vulnerability State:OBSERVED';

describe('useFetchImageCount', () => {
    beforeEach(() => {
        mockedFetchImagesCount.mockReset();
    });

    it('returns the REST image count for the given query', async () => {
        vi.useFakeTimers({ shouldAdvanceTime: true });
        mockedFetchImagesCount.mockResolvedValue(7);

        const { result } = renderHook(() => useFetchImageCount(scopedQuery));

        expect(result.current.isLoading).toBe(true);
        expect(result.current.data).toBeUndefined();
        expect(result.current.error).toBeUndefined();

        await waitForNextUpdate(result);

        expect(result.current.isLoading).toBe(false);
        expect(result.current.data).toBe(7);
        expect(result.current.error).toBeUndefined();
        expect(mockedFetchImagesCount).toHaveBeenCalledWith(scopedQuery);
    });

    it('returns a zero count', async () => {
        vi.useFakeTimers({ shouldAdvanceTime: true });
        mockedFetchImagesCount.mockResolvedValue(0);

        const { result } = renderHook(() => useFetchImageCount());

        await waitForNextUpdate(result);

        expect(result.current.isLoading).toBe(false);
        expect(result.current.data).toBe(0);
        expect(result.current.error).toBeUndefined();
        expect(mockedFetchImagesCount).toHaveBeenCalledWith('');
    });

    it('returns the REST error', async () => {
        vi.useFakeTimers({ shouldAdvanceTime: true });
        const requestError = new Error('count failed');
        mockedFetchImagesCount.mockRejectedValue(requestError);

        const { result } = renderHook(() => useFetchImageCount(scopedQuery));

        await waitForNextUpdate(result);

        expect(result.current.isLoading).toBe(false);
        expect(result.current.data).toBeUndefined();
        expect(result.current.error).toBe(requestError);
    });
});
