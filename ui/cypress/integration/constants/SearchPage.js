const selectors = {
    searchBtn: 'button:contains("Search")',
    pageSearchSuggestions: 'div.Select-menu-outer',
    categoryTabs: '.tab',
    searchInput: '.search-modal div.Select-input > input',
    pageSearchInput: 'div.Select-input > input',
    searchResultsHeader: '.bg-white.flex-1 > .text-xl',
    viewOnViolationsChip: ':nth-child(1) > :nth-child(3) > .p-0 > li > .inline-block',
    viewOnRiskChip: ':nth-child(2) > :nth-child(3) > .p-0 > li:last-child > .inline-block',
    viewOnPoliciesChip: ':nth-child(3) > :nth-child(3) > .p-0 > li > .inline-block',
    viewOnImagesChip: ':nth-child(4) > :nth-child(3) > .p-0 > li > .inline-block'
};

export default selectors;
