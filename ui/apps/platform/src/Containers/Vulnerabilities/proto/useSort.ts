import { useState } from 'react';

/**
 * Manages table sort state keyed by column index, mapped to API sortBy keys.
 *
 * @param columns - Ordered list of API sortBy key strings, one per sortable column.
 * @param defaultIndex - Index into `columns` for the initial sort column.
 * @param defaultDir - Initial sort direction.
 */
export function useSort(
    columns: string[],
    defaultIndex: number,
    defaultDir: 'asc' | 'desc' = 'desc'
) {
    const [activeSortIndex, setActiveSortIndex] = useState(defaultIndex);
    const [activeSortDirection, setActiveSortDirection] = useState<'asc' | 'desc'>(defaultDir);

    const sortBy = columns[activeSortIndex] ?? columns[0];
    const sortDir = activeSortDirection;

    function getThSortProps(columnIndex: number) {
        return {
            sort: {
                sortBy: { index: activeSortIndex, direction: activeSortDirection },
                onSort: (_event: unknown, index: number, direction: 'asc' | 'desc') => {
                    setActiveSortIndex(index);
                    setActiveSortDirection(direction);
                },
                columnIndex,
            },
        };
    }

    return { sortBy, sortDir, activeSortIndex, activeSortDirection, getThSortProps };
}
