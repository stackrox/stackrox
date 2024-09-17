import React from 'react';
import { renderHook, act } from '@testing-library/react';
import { Router } from 'react-router-dom';
import { createMemoryHistory } from 'history';
import cloneDeep from 'lodash/cloneDeep';

import useURLSort from './useURLSort';

const params = {
    sortFields: ['Name', 'Status'],
    defaultSortOption: {
        field: 'Name',
        direction: 'desc',
    },
};

const wrapper = ({ children }) => {
    const history = createMemoryHistory();

    return <Router history={history}>{children}</Router>;
};

beforeAll(() => {
    jest.useFakeTimers();
});

function actAndRunTicks(callback) {
    return act(() => {
        callback();
        jest.runAllTicks();
    });
}

describe('useURLSort', () => {
    describe('when using URL sort with single sort options', () => {
        it('should get the sort options from URL by default', () => {
            const { result } = renderHook(() => useURLSort(params), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);
        });

        it('should change sorting directions on a single field', () => {
            const { result } = renderHook(() => useURLSort(params), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);

            actAndRunTicks(() => {
                result.current.getSortParams('Name').onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(false);

            actAndRunTicks(() => {
                result.current.getSortParams('Name').onSort(null, null, 'desc');
            });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);
        });

        it('should change sorting fields and directions', () => {
            const { result } = renderHook(() => useURLSort(params), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);

            actAndRunTicks(() => {
                result.current.getSortParams('Status').onSort(null, null, 'desc');
            });

            expect(result.current.sortOption.field).toEqual('Status');
            expect(result.current.sortOption.reversed).toEqual(true);

            actAndRunTicks(() => {
                result.current.getSortParams('Status').onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.field).toEqual('Status');
            expect(result.current.sortOption.reversed).toEqual(false);

            actAndRunTicks(() => {
                result.current.getSortParams('Name').onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(false);
        });

        it('should trigger the `onSort` callback when the sort option changes', () => {
            const onSort = jest.fn();

            const { result } = renderHook(() => useURLSort({ ...params, onSort }), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);

            actAndRunTicks(() => {
                result.current.getSortParams('Status').onSort(null, null, 'desc');
            });

            expect(onSort).toHaveBeenCalledTimes(1);
            expect(onSort).toHaveBeenCalledWith({ field: 'Status', direction: 'desc' });
            expect(result.current.sortOption.field).toEqual('Status');
            expect(result.current.sortOption.reversed).toEqual(true);
        });

        it('should retain the passed `aggregateBy` value when sorting', () => {
            const sortParams = cloneDeep(params);
            const aggregateBy = { distinct: true, aggregateFunc: 'count' };
            sortParams.defaultSortOption.aggregateBy = aggregateBy;
            const { result } = renderHook(() => useURLSort(sortParams), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);
            expect(result.current.sortOption.aggregateBy.distinct).toEqual(true);
            expect(result.current.sortOption.aggregateBy.aggregateFunc).toEqual('count');

            actAndRunTicks(() => {
                result.current.getSortParams('Name', aggregateBy).onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(false);
            expect(result.current.sortOption.aggregateBy.distinct).toEqual(true);
            expect(result.current.sortOption.aggregateBy.aggregateFunc).toEqual('count');

            actAndRunTicks(() => {
                result.current.getSortParams('Status').onSort(null, null, 'desc');
            });

            expect(result.current.sortOption.field).toEqual('Status');
            expect(result.current.sortOption.reversed).toEqual(true);
            expect(result.current.sortOption.aggregateBy).toEqual(undefined);
        });

        it('should return the correct PatternFly sort parameters via the `getSortParams` function', () => {
            const { result } = renderHook(() => useURLSort(params), { wrapper });

            // Test handling of both provided fields, and a bogus field that doesn't exist in the sortFields array

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);
            expect(result.current.getSortParams('Name').columnIndex).toEqual(0);
            expect(result.current.getSortParams('Name').sortBy.index).toEqual(0);
            expect(result.current.getSortParams('Name').sortBy.direction).toEqual('desc');
            expect(result.current.getSortParams('Status').columnIndex).toEqual(1);
            expect(result.current.getSortParams('Status').sortBy.index).toEqual(0);
            expect(result.current.getSortParams('Status').sortBy.direction).toEqual('desc');
            expect(result.current.getSortParams('Bogus').columnIndex).toEqual(undefined);
            expect(result.current.getSortParams('Bogus').sortBy.index).toEqual(0);
            expect(result.current.getSortParams('Bogus').sortBy.direction).toEqual('desc');

            actAndRunTicks(() => {
                result.current.getSortParams('Status').onSort(null, null, 'desc');
            });

            expect(result.current.sortOption.field).toEqual('Status');
            expect(result.current.sortOption.reversed).toEqual(true);
            expect(result.current.getSortParams('Name').columnIndex).toEqual(0);
            expect(result.current.getSortParams('Name').sortBy.index).toEqual(1);
            expect(result.current.getSortParams('Name').sortBy.direction).toEqual('desc');
            expect(result.current.getSortParams('Status').columnIndex).toEqual(1);
            expect(result.current.getSortParams('Status').sortBy.index).toEqual(1);
            expect(result.current.getSortParams('Status').sortBy.direction).toEqual('desc');
            expect(result.current.getSortParams('Bogus').columnIndex).toEqual(undefined);
            expect(result.current.getSortParams('Bogus').sortBy.index).toEqual(1);
            expect(result.current.getSortParams('Bogus').sortBy.direction).toEqual('desc');

            actAndRunTicks(() => {
                result.current.getSortParams('Bogus').onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.field).toEqual('Bogus');
            expect(result.current.sortOption.reversed).toEqual(false);
            expect(result.current.getSortParams('Name').columnIndex).toEqual(0);
            expect(result.current.getSortParams('Name').sortBy.index).toEqual(undefined);
            expect(result.current.getSortParams('Name').sortBy.direction).toEqual('asc');
            expect(result.current.getSortParams('Status').columnIndex).toEqual(1);
            expect(result.current.getSortParams('Status').sortBy.index).toEqual(undefined);
            expect(result.current.getSortParams('Status').sortBy.direction).toEqual('asc');
            expect(result.current.getSortParams('Bogus').columnIndex).toEqual(undefined);
            expect(result.current.getSortParams('Bogus').sortBy.index).toEqual(undefined);
            expect(result.current.getSortParams('Bogus').sortBy.direction).toEqual('asc');
        });
    });
});
