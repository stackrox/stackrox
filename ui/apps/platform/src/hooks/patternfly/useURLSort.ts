import { useEffect, useState } from 'react';
import { ThProps } from '@patternfly/react-table';
import useURLParameter from 'hooks/useURLParameter';

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
    sortOption: {
        field: string;
        reversed: boolean;
    };
    getSortParams: GetSortParams;
};

function useURLSort({ sortFields, defaultSortOption }: UseTableSortProps): UseTableSortResult {
    const [sortOption, setSortOption] = useURLParameter<SortOption>(
        'sortOption',
        defaultSortOption
    );

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
        const index = fieldToIndexMap[field];
        const activeSortIndex = activeSortField ? fieldToIndexMap[activeSortField] : undefined;

        return {
            sortBy: {
                index: activeSortIndex,
                direction: activeSortDirection,
                defaultDirection: 'desc',
            },
            onSort: (_event, _index, direction) => {
                // modify the URL based on the new sort
                const newSortOption: SortOption = {
                    field,
                    direction,
                };
                setSortOption(newSortOption);
            },
            columnIndex: index,
        };
    }

    return {
        sortOption: {
            field: activeSortField,
            reversed: activeSortDirection === 'desc',
        },
        getSortParams,
    };
}

export default useURLSort;
