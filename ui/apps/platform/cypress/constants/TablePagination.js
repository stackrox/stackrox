import navigationSelectors from '../selectors/navigation';

export const url = '/main/policies';

export const selectors = {
    configure: `${navigationSelectors.navExpandable}:contains("Platform Configuration")`,
    navLink: `${navigationSelectors.navLinks}:contains("System Policies")`,
    paginationHeader: '[data-testid="pagination-header"]',
    prevPageButton: '[aria-label="Go to previous page"]',
    nextPageButton: '[aria-label="Go to next page"]',
    pageNumberInput: '[data-testid="page-number-input"]',
    tableFirstRow: '.rt-tr-group:first-child .rt-tr',
};
