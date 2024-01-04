import { visit } from '../../../helpers/visit';
import { selectors } from '../workloadCves/WorkloadCves.selectors';

const basePath = '/main/vulnerabilities/exception-management';
const pendingRequestsPath = `${basePath}/pending-requests`;

export function visitExceptionManagement() {
    visit(pendingRequestsPath);

    cy.get('h1:contains("Exception management")');
    cy.location('pathname').should('eq', pendingRequestsPath);
}

/**
 * Select a search filter type from the search filter dropdown.
 * @param {('Request name' | 'CVE' | 'Requester' | 'Image')} searchCategory
 */
export function selectSearchFilterType(entityType) {
    cy.get(selectors.searchOptionsDropdown).click();
    cy.get(selectors.searchOptionsMenuItem(entityType)).click();
    cy.get(selectors.searchOptionsDropdown).click();
}

/**
 * Type a value into the search filter typeahead and select the first matching value.
 * @param {('Request name' | 'CVE' | 'Requester' | 'Image')} searchCategory
 * @param {string} value
 */
export function typeAndEnterSearchFilterValue(entityType, value) {
    selectSearchFilterType(entityType);
    cy.get(selectors.searchOptionsValueTypeahead(entityType)).click();
    cy.get(selectors.searchOptionsValueTypeahead(entityType)).type(`${value}{enter}`);
    // simulates clicking outside the dropdown to make sure it closes
    cy.get('body').click(0, 0);
}
