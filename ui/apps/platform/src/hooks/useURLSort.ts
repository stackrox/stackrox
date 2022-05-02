import { useEffect, useRef, useState } from 'react';
import useURLParameter from 'hooks/useURLParameter';
import { SortDirection, SortOption, ThProps } from 'types/table';
import { ApiSortOption } from 'types/search';
import { isParsedQs } from 'utils/queryStringUtils';

export type GetSortParams = (field: string) => ThProps['sort'] | undefined;

type UseTableSortProps = {
    sortFields: string[];
    defaultSortOption: SortOption;
};

type UseTableSortResult = {
    sortOption: ApiSortOption;
    getSortParams: GetSortParams;
};

function tableSortOption(field: string, direction: SortDirection): ApiSortOption {
    return {
        field,
        reversed: direction === 'desc',
    };
}

function isDirection(val: unknown): val is 'asc' | 'desc' {
    return val === 'asc' || val === 'desc';
}

function useURLSort({ sortFields, defaultSortOption }: UseTableSortProps): UseTableSortResult {
    const [sortOption, setSortOption] = useURLParameter('sortOption', defaultSortOption);

    // get the parsed sort option values from the URL, if available
    // otherwise, use the default sort option values
    const activeSortField =
        isParsedQs(sortOption) && typeof sortOption?.field === 'string'
            ? sortOption.field
            : defaultSortOption.field;
    const activeSortDirection =
        isParsedQs(sortOption) && isDirection(sortOption?.direction)
            ? sortOption.direction
            : defaultSortOption.direction;

    const internalSortResultOption = useRef<ApiSortOption>(
        tableSortOption(activeSortField, activeSortDirection)
    );

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

    if (
        internalSortResultOption.current.field !== activeSortField ||
        internalSortResultOption.current.reversed !== (activeSortDirection === 'desc')
    ) {
        internalSortResultOption.current = tableSortOption(activeSortField, activeSortDirection);
    }

    return {
        sortOption: internalSortResultOption.current,
        getSortParams,
    };
}

export default useURLSort;
