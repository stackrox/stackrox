import React from 'react';
import { renderHook } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import cloneDeep from 'lodash/cloneDeep';

import actAndFlushTaskQueue from 'test-utils/flushTaskQueue';
import useURLSort from './useURLSort';

const wrapper = ({ children }) => {
    return <MemoryRouter>{children}</MemoryRouter>;
};

describe('useURLSort', () => {
    describe('when using URL sort with single sort options', () => {
        const params = {
            sortFields: ['Name', 'Status'],
            defaultSortOption: {
                field: 'Name',
                direction: 'desc',
            },
        };

        it('should get the sort options from URL by default', () => {
            const { result } = renderHook(() => useURLSort(params), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);
        });

        it('should change sorting directions on a single field', async () => {
            const { result } = renderHook(() => useURLSort(params), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);

            await actAndFlushTaskQueue(() => {
                result.current.getSortParams('Name').onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(false);

            await actAndFlushTaskQueue(() => {
                result.current.getSortParams('Name').onSort(null, null, 'desc');
            });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);
        });

        it('should change sorting fields and directions', async () => {
            const { result } = renderHook(() => useURLSort(params), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);

            await actAndFlushTaskQueue(() => {
                result.current.getSortParams('Status').onSort(null, null, 'desc');
            });

            expect(result.current.sortOption.field).toEqual('Status');
            expect(result.current.sortOption.reversed).toEqual(true);

            await actAndFlushTaskQueue(() => {
                result.current.getSortParams('Status').onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.field).toEqual('Status');
            expect(result.current.sortOption.reversed).toEqual(false);

            await actAndFlushTaskQueue(() => {
                result.current.getSortParams('Name').onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(false);
        });

        it('should trigger the `onSort` callback when the sort option changes', async () => {
            const onSort = vi.fn();

            const { result } = renderHook(() => useURLSort({ ...params, onSort }), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);

            await actAndFlushTaskQueue(() => {
                result.current.getSortParams('Status').onSort(null, null, 'desc');
            });

            expect(onSort).toHaveBeenCalledTimes(1);
            expect(onSort).toHaveBeenCalledWith({ field: 'Status', direction: 'desc' });
            expect(result.current.sortOption.field).toEqual('Status');
            expect(result.current.sortOption.reversed).toEqual(true);
        });

        it('should retain the passed `aggregateBy` value when sorting', async () => {
            const sortParams = cloneDeep(params);
            const aggregateBy = { distinct: 'true', aggregateFunc: 'count' };
            sortParams.defaultSortOption.aggregateBy = aggregateBy;
            const { result } = renderHook(() => useURLSort(sortParams), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);
            expect(result.current.sortOption.aggregateBy.distinct).toEqual(true);
            expect(result.current.sortOption.aggregateBy.aggregateFunc).toEqual('count');

            await actAndFlushTaskQueue(() => {
                result.current.getSortParams('Name', aggregateBy).onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(false);
            expect(result.current.sortOption.aggregateBy.distinct).toEqual(true);
            expect(result.current.sortOption.aggregateBy.aggregateFunc).toEqual('count');

            await actAndFlushTaskQueue(() => {
                result.current
                    .getSortParams('Name', { distinct: 'false', aggregateFunc: 'count' })
                    .onSort(null, null, 'asc');
            });
            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(false);
            expect(result.current.sortOption.aggregateBy.distinct).toEqual(false);
            expect(result.current.sortOption.aggregateBy.aggregateFunc).toEqual('count');

            await actAndFlushTaskQueue(() => {
                result.current.getSortParams('Status').onSort(null, null, 'desc');
            });

            expect(result.current.sortOption.field).toEqual('Status');
            expect(result.current.sortOption.reversed).toEqual(true);
            expect(result.current.sortOption.aggregateBy).toEqual(undefined);
        });

        it('should return the correct PatternFly sort parameters via the `getSortParams` function', async () => {
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
            expect(result.current.getSortParams('Bogus').columnIndex).toEqual(-1);
            expect(result.current.getSortParams('Bogus').sortBy.index).toEqual(0);
            expect(result.current.getSortParams('Bogus').sortBy.direction).toEqual('desc');

            await actAndFlushTaskQueue(() => {
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
            expect(result.current.getSortParams('Bogus').columnIndex).toEqual(-1);
            expect(result.current.getSortParams('Bogus').sortBy.index).toEqual(1);
            expect(result.current.getSortParams('Bogus').sortBy.direction).toEqual('desc');

            await actAndFlushTaskQueue(() => {
                result.current.getSortParams('Bogus').onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.field).toEqual('Bogus');
            expect(result.current.sortOption.reversed).toEqual(false);
            expect(result.current.getSortParams('Name').columnIndex).toEqual(0);
            expect(result.current.getSortParams('Name').sortBy.index).toEqual(-1);
            expect(result.current.getSortParams('Name').sortBy.direction).toEqual('asc');
            expect(result.current.getSortParams('Status').columnIndex).toEqual(1);
            expect(result.current.getSortParams('Status').sortBy.index).toEqual(-1);
            expect(result.current.getSortParams('Status').sortBy.direction).toEqual('asc');
            expect(result.current.getSortParams('Bogus').columnIndex).toEqual(-1);
            expect(result.current.getSortParams('Bogus').sortBy.index).toEqual(-1);
            expect(result.current.getSortParams('Bogus').sortBy.direction).toEqual('asc');
        });
    });

    describe('when using URL sort with multiple sort options', () => {
        const params = {
            sortFields: [
                'Name',
                'Status',
                ['Critical severity count', 'Low severity count'],
                'Created date',
            ],
            defaultSortOption: {
                field: 'Name',
                direction: 'desc',
            },
        };

        it('should get the sort options from URL by default', () => {
            const { result } = renderHook(() => useURLSort(params), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);
        });

        it('should return the default sort option when an empty multi field sort option is provided', async () => {
            const { result } = renderHook(() => useURLSort(params), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);

            await actAndFlushTaskQueue(() => {
                result.current.getSortParams('Severity', []).onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);
        });

        it('should change sorting fields and directions', async () => {
            const { result } = renderHook(() => useURLSort(params), { wrapper });

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);

            // Change to single field sort
            await actAndFlushTaskQueue(() => {
                result.current.getSortParams('Status').onSort(null, null, 'desc');
            });

            expect(result.current.sortOption.field).toEqual('Status');
            expect(result.current.sortOption.reversed).toEqual(true);

            // Change to subset of multi field sort
            await actAndFlushTaskQueue(() => {
                result.current
                    .getSortParams('Severity', [{ field: 'Critical severity count' }])
                    .onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.length).toEqual(1);
            expect(result.current.sortOption[0].field).toEqual('Critical severity count');
            expect(result.current.sortOption[0].reversed).toEqual(false);

            // A multi field sort option that matches a single field sort field will match
            // and return the sort option in array form
            await actAndFlushTaskQueue(() => {
                result.current.getSortParams('Name', [{ field: 'Name' }]).onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.length).toEqual(1);
            expect(result.current.sortOption[0].field).toEqual('Name');
            expect(result.current.sortOption[0].reversed).toEqual(false);

            // Change to multi field sort with aggregateBy using multi sort function parameters
            await actAndFlushTaskQueue(() => {
                result.current
                    .getSortParams('Severity', [
                        {
                            field: 'Critical severity count',
                            aggregateBy: { distinct: 'true', aggregateFunc: 'max' },
                        },
                        {
                            field: 'Low severity count',
                            aggregateBy: { distinct: 'true', aggregateFunc: 'max' },
                        },
                    ])
                    .onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.length).toEqual(2);
            expect(result.current.sortOption[0].field).toEqual('Critical severity count');
            expect(result.current.sortOption[0].reversed).toEqual(false);
            expect(result.current.sortOption[0].aggregateBy.distinct).toEqual(true);
            expect(result.current.sortOption[0].aggregateBy.aggregateFunc).toEqual('max');
            expect(result.current.sortOption[1].field).toEqual('Low severity count');
            expect(result.current.sortOption[1].reversed).toEqual(false);
            expect(result.current.sortOption[1].aggregateBy.distinct).toEqual(true);
            expect(result.current.sortOption[1].aggregateBy.aggregateFunc).toEqual('max');
        });

        it('should return the correct PatternFly sort parameters via the `getSortParams` function', async () => {
            const { result } = renderHook(() => useURLSort(params), { wrapper });

            // Test handling of both provided fields, and a bogus fields that do not exist in the sortFields array

            expect(result.current.sortOption.field).toEqual('Name');
            expect(result.current.sortOption.reversed).toEqual(true);
            expect(result.current.getSortParams('Name').columnIndex).toEqual(0);
            expect(result.current.getSortParams('Name').sortBy.index).toEqual(0);
            expect(result.current.getSortParams('Name').sortBy.direction).toEqual('desc');
            expect(result.current.getSortParams('Status').columnIndex).toEqual(1);
            expect(result.current.getSortParams('Status').sortBy.index).toEqual(0);
            expect(result.current.getSortParams('Status').sortBy.direction).toEqual('desc');
            expect(result.current.getSortParams('Bogus').columnIndex).toEqual(-1);
            expect(result.current.getSortParams('Bogus').sortBy.index).toEqual(0);
            expect(result.current.getSortParams('Bogus').sortBy.direction).toEqual('desc');
            const criticalSeveritySortParams = result.current.getSortParams('Severity', [
                { field: 'Critical severity count' },
            ]);
            expect(criticalSeveritySortParams.columnIndex).toEqual(2);
            expect(criticalSeveritySortParams.sortBy.index).toEqual(0);
            expect(criticalSeveritySortParams.sortBy.direction).toEqual('desc');
            const bogusSeveritySortParams = result.current.getSortParams('Severity', [
                { field: 'Bogus severity count' },
            ]);
            expect(bogusSeveritySortParams.columnIndex).toEqual(-1);
            expect(bogusSeveritySortParams.sortBy.index).toEqual(0);
            expect(bogusSeveritySortParams.sortBy.direction).toEqual('desc');

            await actAndFlushTaskQueue(() => {
                result.current
                    .getSortParams('Severity', [{ field: 'Critical severity count' }])
                    .onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.length).toEqual(1);
            expect(result.current.sortOption[0].field).toEqual('Critical severity count');
            expect(result.current.sortOption[0].reversed).toEqual(false);
            expect(result.current.getSortParams('Name').columnIndex).toEqual(0);
            expect(result.current.getSortParams('Name').sortBy.index).toEqual(2);
            expect(result.current.getSortParams('Name').sortBy.direction).toEqual('asc');
            expect(result.current.getSortParams('Status').columnIndex).toEqual(1);
            expect(result.current.getSortParams('Status').sortBy.index).toEqual(2);
            expect(result.current.getSortParams('Status').sortBy.direction).toEqual('asc');
            expect(result.current.getSortParams('Bogus').columnIndex).toEqual(-1);
            expect(result.current.getSortParams('Bogus').sortBy.index).toEqual(2);
            expect(result.current.getSortParams('Bogus').sortBy.direction).toEqual('asc');
            expect(
                result.current.getSortParams('Severity', [{ field: 'Critical severity count' }])
                    .columnIndex
            ).toEqual(2);
            expect(
                result.current.getSortParams('Severity', [{ field: 'Critical severity count' }])
                    .sortBy.index
            ).toEqual(2);
            expect(
                result.current.getSortParams('Severity', [{ field: 'Critical severity count' }])
                    .sortBy.direction
            ).toEqual('asc');

            await actAndFlushTaskQueue(() => {
                result.current
                    .getSortParams('Bogus', [{ field: 'Bogus severity count' }])
                    .onSort(null, null, 'asc');
            });

            expect(result.current.sortOption.length).toEqual(1);
            expect(result.current.sortOption[0].field).toEqual('Bogus severity count');
            expect(result.current.sortOption[0].reversed).toEqual(false);
            expect(result.current.getSortParams('Name').columnIndex).toEqual(0);
            expect(result.current.getSortParams('Name').sortBy.index).toEqual(-1);
            expect(result.current.getSortParams('Name').sortBy.direction).toEqual('asc');
            expect(result.current.getSortParams('Status').columnIndex).toEqual(1);
            expect(result.current.getSortParams('Status').sortBy.index).toEqual(-1);
            expect(result.current.getSortParams('Status').sortBy.direction).toEqual('asc');
            expect(result.current.getSortParams('Bogus').columnIndex).toEqual(-1);
            expect(result.current.getSortParams('Bogus').sortBy.index).toEqual(-1);
            expect(result.current.getSortParams('Bogus').sortBy.direction).toEqual('asc');
            expect(
                result.current.getSortParams('Severity', [{ field: 'Critical severity count' }])
                    .columnIndex
            ).toEqual(2);
            expect(
                result.current.getSortParams('Severity', [{ field: 'Critical severity count' }])
                    .sortBy.index
            ).toEqual(-1);
            expect(
                result.current.getSortParams('Severity', [{ field: 'Critical severity count' }])
                    .sortBy.direction
            ).toEqual('asc');
        });
    });
});
