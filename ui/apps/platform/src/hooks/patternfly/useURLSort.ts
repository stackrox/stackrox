import { useEffect, useState } from 'react';
import { useLocation, useHistory } from 'react-router-dom';
import { ThProps } from '@patternfly/react-table';

import { getQueryObject, getQueryString } from 'utils/queryStringUtils';

export type SortOption = {
    field: string;
    direction: 'asc' | 'desc';
};

type SearchObject = {
    sortOption?: SortOption;
};

type SortOptionFilter = SortOption | undefined;

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
    const history = useHistory();
    const location = useLocation();
    const sortOptionFilter: SortOptionFilter = getQueryObject<SearchObject>(
        location.search
    )?.sortOption;

    // get the sort option values from the URL, if available
    // otherwise, use the default sort option values
    const activeSortField = sortOptionFilter?.field || defaultSortOption.field;
    const activeSortDirection = sortOptionFilter?.direction || defaultSortOption.direction;

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
                const querySearchObject = getQueryObject<Record<string, string | string[]>>(
                    location.search
                );
                const newSortOptionString = getQueryString({
                    ...querySearchObject,
                    sortOption: {
                        field,
                        direction,
                    },
                });
                history.replace({
                    search: newSortOptionString,
                });
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
