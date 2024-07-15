const filterChipSection =
    '.pf-v5-c-toolbar .pf-v5-c-toolbar__group[aria-label="applied search filters"]';

export const selectors = {
    clearFiltersButton: `${filterChipSection} button:contains("Clear filters")`,
    entityTypeToggleItem: (entityType: string) =>
        `.pf-v5-c-toggle-group[aria-label="Entity type toggle items"] button:contains("${entityType}")`,
} as const;
