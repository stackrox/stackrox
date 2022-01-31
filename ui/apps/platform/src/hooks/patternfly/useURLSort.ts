import { useEffect, useState } from 'react';
import { useLocation, useHistory } from 'react-router-dom';
import { ThProps } from '@patternfly/react-table';

import { getQueryObject, getQueryString } from 'utils/queryStringUtils';
import useURLSearchState from './useURLSearchState';

export type SortOption = {
    field: string;
    direction: 'asc' | 'desc';
};

export type GetSortParams = (field: string) => ThProps['sort'] | undefined;

type UseTableSortProps = {
    sortFields: string[];
    defaultSortOption: SortOption;
};

type UseTableSortResult = {
    sortOption: SortOption;
    getSortParams: GetSortParams;
};

function useURLSort({ sortFields, defaultSortOption }: UseTableSortProps): UseTableSortResult {
    const [sortOption, setSortOption] = useURLSearchState<SortOption>('sortOption');

    // get the sort option values from the URL, if available
    // otherwise, use the default sort option values
    const activeSortField = sortOption?.field || defaultSortOption.field;
    const activeSortDirection = sortOption?.direction || defaultSortOption.direction;

    // we'll use this to map the sort fields to an index PatternFly can use internally
    const [fieldToIndexMap, setFieldToIndexMap] = useState<Record<string, number>>({});

    // we'll construct a map of sort fields to indices that will make it easier to work with
    // PatternFly
    useEffect(() => {
        const newFieldToIndexMap = sortFields.reduce((acc, curr, index) => {
            acc[curr] = index;
            return acc;
        }, {} as Record<string, number>);
        setFieldToIndexMap(newFieldToIndexMap);
    }, [sortFields]);

    function getSortParams(field: string): ThProps['sort'] {
        const columnIndex = fieldToIndexMap[field];
        const activeSortIndex = activeSortField ? fieldToIndexMap[activeSortField] : undefined;

        return {
            sortBy: {
                index: activeSortIndex,
                direction: activeSortDirection,
                defaultDirection: 'asc',
            },
            onSort: (_event, _index, direction) => {
                // modify the URL based on the new sort
                const newSortOption: SortOption = {
                    field,
                    direction
                }
                setSortOption(newSortOption);
            },
            columnIndex,
        };
    }

    return {
        sortOption: {
            field: activeSortField,
            direction: activeSortDirection,
        },
        getSortParams,
    };
}

export default useURLSort;
