export const selectors = {
    tableSortColumn: (columnName: string) => `table th.pf-c-table__sort:contains("${columnName}")`,
    tableColumnSortButton: (columnName: string) =>
        `table th.pf-c-table__sort button:contains("${columnName}")`,
};
