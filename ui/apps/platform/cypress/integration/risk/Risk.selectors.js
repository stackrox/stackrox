import pf6 from '../../selectors/pf6';

export const selectors = {
    search: {
        // Default compound search filter input (Deployment entity, Name attribute).
        input: '[aria-label="Filter results by Deployment name"]',
        // Container for applied filter chips rendered by CompoundSearchFilterLabels.
        filterChips: '.search-filter-labels',
        filterChipGroup: `.search-filter-labels ${pf6.labelGroup}`,
    },
    createPolicyButton: 'button:contains("Create policy")',
};
