import React from 'react';
import { renderHook, act } from '@testing-library/react';
import { Router } from 'react-router-dom';
import { createMemoryHistory } from 'history';

import useURLSort from './useURLSort';

const history = createMemoryHistory();

const params = {
    sortFields: ['Name', 'Status'],
    defaultSortOption: {
        field: 'Name',
        direction: 'desc',
    },
};

const wrapper = ({ children }) => {
    return <Router history={history}>{children}</Router>;
};

describe('useURLSort', () => {
    it('should get the sort options from URL by default', () => {
        const { result } = renderHook(
            () => {
                return useURLSort(params);
            },
            {
                wrapper,
            }
        );

        expect(result.current.sortOption.field).toEqual('Name');
        expect(result.current.sortOption.reversed).toEqual(true);
    });

    it('should keep sorting to the "Status" field and change direction to "asc"', () => {
        const { result } = renderHook(
            () => {
                return useURLSort(params);
            },
            {
                wrapper,
            }
        );

        act(() => {
            result.current.getSortParams('Name').onSort(null, null, 'asc');
        });

        expect(result.current.sortOption.field).toEqual('Name');
        expect(result.current.sortOption.reversed).toEqual(false);
    });

    it('should change sorting to the "Status" field and direction to "desc"', () => {
        const { result } = renderHook(
            () => {
                return useURLSort(params);
            },
            {
                wrapper,
            }
        );

        act(() => {
            result.current.getSortParams('Status').onSort(null, null, 'desc');
        });

        expect(result.current.sortOption.field).toEqual('Status');
        expect(result.current.sortOption.reversed).toEqual(true);
    });
});
