import { selectors } from './vulnerabilities.selectors';

export function clearSearchFilters() {
    cy.get('body').then((body) => {
        if (body.find(selectors.clearFiltersButton).length > 0) {
            // If button exists, click it
            cy.get(selectors.clearFiltersButton).click(); // Note: This is a workaround to prevent a lack of CVE data from causing the test to fail in CI
        }
    });
}
