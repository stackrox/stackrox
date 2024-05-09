const filterChipSection =
    '.pf-v5-c-toolbar .pf-v5-c-toolbar__group[aria-label="applied search filters"]';

export const selectors = {
    clearFiltersButton: `${filterChipSection} button:contains("Clear filters")`,
} as const;
