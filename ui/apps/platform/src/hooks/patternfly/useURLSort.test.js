import React from 'react';
import { renderHook, act } from '@testing-library/react-hooks';
import { Router } from 'react-router-dom';
import { createMemoryHistory } from 'history';

import useURLSort from './useURLSort';

const history = createMemoryHistory();

const sortFields = ['Name', 'Status'];

describe('useURLSort', () => {
    it('should get the sort options from URL by default', () => {
        const { result } = renderHook(
            () => {
                return useURLSort({
                    sortFields,
                    defaultSortOption: {
                        field: 'Name',
                        direction: 'desc',
                    },
                });
            },
            {
                wrapper: ({ children }) => {
                    return <Router history={history}>{children}</Router>;
                },
            }
        );

        expect(result.current.sortOption.field).toEqual('Name');
        expect(result.current.sortOption.direction).toEqual('desc');
    });

    it('should keep sorting to the "Status" field and change direction to "asc"', () => {
        const { result } = renderHook(
            () => {
                return useURLSort({
                    sortFields,
                    defaultSortOption: {
                        field: 'Name',
                        direction: 'desc',
                    },
                });
            },
            {
                wrapper: ({ children }) => {
                    return <Router history={history}>{children}</Router>;
                },
            }
        );

        act(() => {
            result.current.getSortParams('Name').onSort(null, null, 'asc');
        });

        expect(result.current.sortOption.field).toEqual('Name');
        expect(result.current.sortOption.direction).toEqual('asc');
    });

    it('should change sorting to the "Status" field and direction to "desc"', () => {
        const { result } = renderHook(
            () => {
                return useURLSort({
                    sortFields,
                    defaultSortOption: {
                        field: 'Name',
                        direction: 'desc',
                    },
                });
            },
            {
                wrapper: ({ children }) => {
                    return <Router history={history}>{children}</Router>;
                },
            }
        );

        act(() => {
            result.current.getSortParams('Status').onSort(null, null, 'desc');
        });

        expect(result.current.sortOption.field).toEqual('Status');
        expect(result.current.sortOption.direction).toEqual('desc');
    });
});
