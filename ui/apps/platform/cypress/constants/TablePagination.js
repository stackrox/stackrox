export const url = '/main/policies';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Platform Configuration") a',
    navLink: '.navigation-panel li:contains("System Policies") a',
    paginationHeader: '[data-testid="pagination-header"]',
    prevPageButton: '[data-testid="prev-page-button"]',
    nextPageButton: '[data-testid="next-page-button"]',
    pageNumberInput: '[data-testid="page-number-input"]',
    tableFirstRow: '.rt-tr-group:first-child .rt-tr',
};
