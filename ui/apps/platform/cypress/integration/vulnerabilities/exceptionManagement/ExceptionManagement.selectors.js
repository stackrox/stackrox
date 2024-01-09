export const selectors = {
    tableSortColumn: (columnName) => `table th.pf-c-table__sort:contains("${columnName}")`,
    tableColumnSortButton: (columnName) =>
        `table th.pf-c-table__sort button:contains("${columnName}")`,
};
