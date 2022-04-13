import { useEffect, useState } from 'react';
import { SortDirection, TableColumn, TableSortOption } from 'types/table';

export type UseTableSort = {
    activeSortIndex: number;
    setActiveSortIndex: (idx) => void;
    activeSortDirection: SortDirection;
    setActiveSortDirection: (dir) => void;
    sortOption: TableSortOption;
};

function useTableSort(columns: TableColumn[], defaultSort: TableSortOption): UseTableSort {
    const defaultSortIndex = columns.findIndex((column) => column?.sortField === defaultSort.field);
    const defaultSortDirection = defaultSort.reversed ? 'desc' : 'asc';
    // index of the currently active column
    const [activeSortIndex, setActiveSortIndex] = useState(defaultSortIndex);
    // sort direction of the currently active column
    const [activeSortDirection, setActiveSortDirection] =
        useState<SortDirection>(defaultSortDirection);

    const [sortOption, setSortOption] = useState<TableSortOption>(defaultSort);

    useEffect(() => {
        const { sortField } = columns[activeSortIndex];
        if (sortField) {
            const newSortOption: TableSortOption = {
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
