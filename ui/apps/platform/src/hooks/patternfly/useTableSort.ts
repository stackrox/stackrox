import { useEffect, useState } from 'react';
import { ThProps } from '@patternfly/react-table';
import { SortOption } from 'types/table';
import { ApiSortOption } from 'types/search';

export type GetSortParams = (field: string) => ThProps['sort'];

type UseTableSortProps = {
    sortFields: string[];
    defaultSortOption: SortOption;
};

type UseTableSortResult = {
    sortOption: ApiSortOption;
    getSortParams: GetSortParams;
};

function useURLSort({ sortFields, defaultSortOption }: UseTableSortProps): UseTableSortResult {
    const [sortOption, setSortOption] = useState<SortOption>();

    // get the sort option values from the URL, if available
    // otherwise, use the default sort option values
    const activeSortField = sortOption?.field || defaultSortOption.field;
    const activeSortDirection = sortOption?.direction || defaultSortOption.direction;

    // we'll use this to map the sort fields to an id PatternFly can use internally
    const [fieldToIdMap, setFieldToIdMap] = useState<Record<string, number>>({});

    // we'll construct a map of sort fields to ids that will make it easier to work with
    // PatternFly
    useEffect(() => {
        const newFieldToIdMap = sortFields.reduce(
            (acc, curr, index) => {
                acc[curr] = index;
                return acc;
            },
            {} as Record<string, number>
        );
        setFieldToIdMap(newFieldToIdMap);
    }, [sortFields]);

    function getSortParams(field: string): ThProps['sort'] {
        const fieldId = fieldToIdMap[field];
        const activeSortId = activeSortField ? fieldToIdMap[activeSortField] : undefined;

        return {
            sortBy: {
                index: activeSortId,
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
            columnIndex: fieldId,
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
