import { useEffect, useRef, useState } from 'react';
import useURLParameter from 'hooks/useURLParameter';
import { SortAggregate, SortDirection, SortOption, ThProps } from 'types/table';
import { ApiSortOption } from 'types/search';
import { isParsedQs } from 'utils/queryStringUtils';

export type GetSortParams = (
    field: string,
    aggregateBy?: SortAggregate
) => ThProps['sort'] | undefined;

export type UseURLSortProps = {
    sortFields: string[];
    defaultSortOption: SortOption;
    onSort?: (newSortOption: SortOption) => void;
};

export type UseURLSortResult = {
    sortOption: ApiSortOption;
    setSortOption: (newSortOption: SortOption) => void;
    getSortParams: GetSortParams;
};

function tableSortOption(
    field: string,
    direction: SortDirection,
    aggregateBy?: SortAggregate
): ApiSortOption {
    const sortOption = {
        field,
        reversed: direction === 'desc',
    };
    if (aggregateBy) {
        const { aggregateFunc, distinct } = aggregateBy;
        return {
            ...sortOption,
            aggregateBy: {
                aggregateFunc,
                distinct: !!distinct,
            },
        };
    }
    return sortOption;
}

function isDirection(val: unknown): val is 'asc' | 'desc' {
    return val === 'asc' || val === 'desc';
}

function useURLSort({ sortFields, defaultSortOption, onSort }: UseURLSortProps): UseURLSortResult {
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
    const activeAggregateBy =
        isParsedQs(sortOption) && sortOption?.aggregateBy
            ? (sortOption?.aggregateBy as SortAggregate)
            : undefined;

    const internalSortResultOption = useRef<ApiSortOption>(
        tableSortOption(activeSortField, activeSortDirection, activeAggregateBy)
    );

    // we'll use this to map the sort fields to an index PatternFly can use internally
    const [fieldToIndexMap, setFieldToIndexMap] = useState<Record<string, number>>({});

    // we'll construct a map of sort fields to indices that will make it easier to work with
    // PatternFly
    useEffect(() => {
        const newFieldToIndexMap = sortFields.reduce(
            (acc, curr, index) => {
                acc[curr] = index;
                return acc;
            },
            {} as Record<string, number>
        );
        setFieldToIndexMap(newFieldToIndexMap);
    }, [sortFields]);

    function getSortParams(field: string, aggregateBy?: SortAggregate): ThProps['sort'] {
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
                    aggregateBy,
                    direction,
                };
                if (onSort) {
                    onSort(newSortOption);
                }
                setSortOption(newSortOption);
            },
            columnIndex: index,
        };
    }

    if (
        internalSortResultOption.current.field !== activeSortField ||
        internalSortResultOption.current.reversed !== (activeSortDirection === 'desc')
    ) {
        internalSortResultOption.current = tableSortOption(
            activeSortField,
            activeSortDirection,
            activeAggregateBy
        );
    }

    return {
        sortOption: internalSortResultOption.current,
        setSortOption: (newSortOption: SortOption) => {
            setSortOption(newSortOption);
        },
        getSortParams,
    };
}

export default useURLSort;
