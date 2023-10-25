import { renderHook, act } from '@testing-library/react';
import { usePaginatedQuery } from 'hooks/usePaginatedQuery';
import waitForNextUpdate from 'test-utils/waitForNextUpdate';

describe('usePaginatedQuery hook', () => {
    const data = [
        { id: '1' },
        { id: '2' },
        { id: '3' },
        { id: '4' },
        { id: '4' }, // Duplicate item to simulate server side data changes or inconsistencies
        { id: '5' },
        { id: '6' },
        { id: '7' },
        { id: '8' },
    ] as const;

    const pageSize = 2;
    const requestFn = (page: number) => {
        const offset = page * pageSize;
        return Promise.resolve(data.slice(offset, offset + pageSize));
    };

    it('should return paginated data without duplicates', async () => {
        const { result } = renderHook(() =>
            usePaginatedQuery(requestFn, pageSize, { dedupKeyFn: ({ id }) => id, debounceRate: 10 })
        );

        // Test initial empty state
        expect(result.current.data).toHaveLength(0);

        // Test after initial automatic fetch
        await waitForNextUpdate(result);
        expect(result.current.data).toEqual([[{ id: '1' }, { id: '2' }]]);

        // Test event causing a fetch of the next page
        act(() => {
            // Test with an `immediate=true` parameter to cover immediate and debounced use cases
            result.current.fetchNextPage(true);
        });
        await waitForNextUpdate(result);
        expect(result.current.data).toEqual([
            [{ id: '1' }, { id: '2' }],
            [{ id: '3' }, { id: '4' }],
        ]);

        // Test event causing a fetch with duplicate data
        act(() => {
            result.current.fetchNextPage();
        });
        await waitForNextUpdate(result);
        expect(result.current.data).toEqual([
            [{ id: '1' }, { id: '2' }],
            [{ id: '3' }, { id: '4' }],
            [{ id: '5' }],
        ]);

        // Test subsequent fetch without duplicates again
        act(() => {
            result.current.fetchNextPage();
        });
        await waitForNextUpdate(result);
        expect(result.current.data).toEqual([
            [{ id: '1' }, { id: '2' }],
            [{ id: '3' }, { id: '4' }],
            [{ id: '5' }],
            [{ id: '6' }, { id: '7' }],
        ]);

        // Test that end of results detected correctly
        expect(result.current.isEndOfResults).toBeFalsy(); // `{ id: '8' }` still remaining
        act(() => {
            result.current.fetchNextPage();
        });
        await waitForNextUpdate(result);
        expect(result.current.isEndOfResults).toBeTruthy();
    });

    it('should clear cached data when reset or clear function is called', async () => {
        const { result } = renderHook(() =>
            usePaginatedQuery(requestFn, pageSize, { dedupKeyFn: ({ id }) => id, debounceRate: 10 })
        );

        // Test initial empty state
        expect(result.current.data).toHaveLength(0);

        // Test after initial automatic fetch
        await waitForNextUpdate(result);
        expect(result.current.data).toEqual([[{ id: '1' }, { id: '2' }]]);

        // Test that data is cleared correctly
        act(() => {
            result.current.resetPages();
        });
        expect(result.current.isRefreshingResults).toBeTruthy();
        expect(result.current.isFetchingNextPage).toBeTruthy();
        expect(result.current.data).toHaveLength(0);

        // Test that reset automatically fetches the first page
        await waitForNextUpdate(result);
        expect(result.current.data).toHaveLength(1);
        expect(result.current.isRefreshingResults).toBeFalsy();
        expect(result.current.isFetchingNextPage).toBeFalsy();

        // Test that clearing the data works correctly and _does not_ automatically fetch a new page
        act(() => {
            result.current.clearPages();
        });
        expect(result.current.data).toHaveLength(0);
        expect(result.current.isRefreshingResults).toBeFalsy();
        expect(result.current.isFetchingNextPage).toBeFalsy();
    });
});
