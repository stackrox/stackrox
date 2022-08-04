import navSelectors from '../selectors/navigation';
import { visitMainDashboard } from './main';
import { interceptRequests, waitForResponses } from './request';

/*
 * For example, visitFromLeftNav('Violations');
 */
export function visitFromLeftNav(itemText, requestConfig) {
    visitMainDashboard();

    interceptRequests(requestConfig);
    cy.get(`${navSelectors.navLinks}:contains("${itemText}")`).click();
    waitForResponses(requestConfig);
}

/*
 * For example, visitFromLeftNavExpandable('Vulnerability Management', 'Reporting');
 * For example, visitFromLeftNavExpandable('Platform Configuration', 'Integrations');
 */
export function visitFromLeftNavExpandable(expandableTitle, itemText, requestConfig) {
    visitMainDashboard();

    interceptRequests(requestConfig);
    cy.get(`${navSelectors.navExpandable}:contains("${expandableTitle}")`).click();
    cy.get(`${navSelectors.nestedNavLinks}:contains("${itemText}")`).click();
    waitForResponses(requestConfig);
}
