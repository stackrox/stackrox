export const url = '/main/policies';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Configure") a',
    navLink: '.navigation-panel li:contains("System Policies") a',
    paginationHeader: '[data-test-id="pagination-header"]',
    prevPageButton: '[data-test-id="prev-page-button"]',
    nextPageButton: '[data-test-id="next-page-button"]',
    pageNumberInput: '[data-test-id="page-number-input"]',
    tableFirstRow: '.rt-tr-group:first-child .rt-tr'
};
