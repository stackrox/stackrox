import navigationSelectors from '../selectors/navigation';

export const url = '/main/policies';

export const selectors = {
    configure: `${navigationSelectors.navExpandable}:contains("Platform Configuration")`,
    navLink: `${navigationSelectors.navLinks}:contains("System Policies")`,
    paginationHeader: '[data-testid="pagination-header"]',
    prevPageButton: '[data-testid="prev-page-button"]',
    nextPageButton: '[data-testid="next-page-button"]',
    pageNumberInput: '[data-testid="page-number-input"]',
    tableFirstRow: '.rt-tr-group:first-child .rt-tr',
};
