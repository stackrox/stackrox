export const selectors = {
    tableSortColumn: (columnName: string) =>
        `table th.pf-c-table__sort:contains("${columnName}")` as const,
    tableColumnSortButton: (columnName: string) =>
        `table th.pf-c-table__sort button:contains("${columnName}")` as const,
} as const;
