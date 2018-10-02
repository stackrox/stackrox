const selectors = {
    panelHeader: 'div[data-test-id="panel"]',
    searchBtn: 'button:contains("Search")',
    pageSearchSuggestions: 'div.Select-menu-outer',
    categoryTabs: '.tab',
    searchInput: '.search-modal div.Select-input > input',
    pageSearchInput: 'div.Select-input > input',
    searchResultsHeader: '.bg-base-100.flex-1 > .text-xl',
    viewOnViolationsChip:
        'div.rt-tbody > .rt-tr-group:first-child .rt-tr .rt-td:nth-child(3) ul > li:first-child > button',
    viewOnRiskChip:
        'div.rt-tbody > .rt-tr-group:nth-child(2) .rt-tr .rt-td:nth-child(3) ul > li:first-child > button',
    viewOnPoliciesChip:
        'div.rt-tbody > .rt-tr-group:nth-child(3) .rt-tr .rt-td:nth-child(3) ul > li:first-child > button ',
    viewOnImagesChip:
        'div.rt-tbody > .rt-tr-group:nth-child(4) .rt-tr .rt-td:nth-child(3) ul > li:first-child > button'
};

export default selectors;
