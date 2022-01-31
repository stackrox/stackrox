import React from 'react';
import { renderHook } from '@testing-library/react-hooks';
import { Router } from 'react-router-dom';
import { createMemoryHistory } from 'history';

import useURLPagination from './useURLPagination';

const history = createMemoryHistory();

const wrapper = ({ children }) => {
    return <Router history={history}>{children}</Router>;
};

describe('useURLPagination', () => {
    it('should get the default pagination values', () => {
        const { result } = renderHook(
            () => {
                return useURLPagination();
            },
            {
                wrapper,
            }
        );

        expect(result.current.page).toEqual(1);
        expect(result.current.perPage).toEqual(20);
    });

    // @TODO: Add more tests
});
