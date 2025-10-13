const filterChipSection =
    '.pf-v5-c-toolbar .pf-v5-c-toolbar__group[aria-label="applied search filters"]';

export const selectors = {
    clearFiltersButton: `${filterChipSection} button:contains("Clear filters")`,
    entityTypeToggleItem: (entityType: string) =>
        `.pf-v5-c-toggle-group[aria-label="Entity type toggle items"] button:contains("${entityType}")`,
    summaryCard: (cardTitle) => `.pf-v5-c-card:contains("${cardTitle}")`,

    expandRowButton: 'table tbody tr button[aria-label="Details"]',
    expandableRow: 'table tbody tr.pf-v5-c-table__expandable-row',
} as const;
