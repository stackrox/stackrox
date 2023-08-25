import { useEffect, useState } from 'react';
import { SortDirection, TableColumn } from 'types/table';
import { ApiSortOption } from 'types/search';

export type UseTableSort = {
    activeSortIndex: number;
    setActiveSortIndex: (idx) => void;
    activeSortDirection: SortDirection;
    setActiveSortDirection: (dir) => void;
    sortOption: ApiSortOption;
};

function useTableSort(columns: TableColumn[], defaultSort: ApiSortOption): UseTableSort {
    const defaultSortIndex = columns.findIndex((column) => column?.sortField === defaultSort.field);
    const defaultSortDirection = defaultSort.reversed ? 'desc' : 'asc';
    // index of the currently active column
    const [activeSortIndex, setActiveSortIndex] = useState(defaultSortIndex);
    // sort direction of the currently active column
    const [activeSortDirection, setActiveSortDirection] =
        useState<SortDirection>(defaultSortDirection);

    const [sortOption, setSortOption] = useState<ApiSortOption>(defaultSort);

    useEffect(() => {
        const { sortField } = columns[activeSortIndex];
        if (sortField) {
            const newSortOption: ApiSortOption = {
                field: sortField,
                reversed: activeSortDirection === 'desc',
            };
            setSortOption(newSortOption);
        }
    }, [activeSortIndex, activeSortDirection, columns]);

    return {
        activeSortIndex,
        setActiveSortIndex,
        activeSortDirection,
        setActiveSortDirection,
        sortOption,
    };
}

export default useTableSort;
