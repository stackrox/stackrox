export const selectors = {
    tableSortColumn: (columnName: string) =>
        `table th.pf-c-table__sort:contains("${columnName}")` as const,
    tableColumnSortButton: (columnName: string) =>
        `table th.pf-c-table__sort button:contains("${columnName}")` as const,
    approvedDeferralsTab: 'button[role="tab"]:contains("Approved deferrals")',
    approvedFalsePositivesTab: 'button[role="tab"]:contains("Approved false positives")',
    deniedRequestsTab: 'button[role="tab"]:contains("Denied requests")',
} as const;
