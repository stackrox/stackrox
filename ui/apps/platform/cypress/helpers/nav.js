import navSelectors from '../selectors/navigation';
import { visitMainDashboard } from './main';

/*
 * For example, visitFromLeftNav('Violations');
 */
export function visitFromLeftNav(itemText) {
    visitMainDashboard();
    cy.get(`${navSelectors.navLinks}:contains("${itemText}")`).click();
}

/*
 * For example, visitFromLeftNavExpandable('Vulnerability Management', 'Reporting');
 * For example, visitFromLeftNavExpandable('Platform Configuration', 'Integrations');
 */
export function visitFromLeftNavExpandable(expandableTitle, itemText) {
    visitMainDashboard();
    cy.get(`${navSelectors.navExpandable}:contains("${expandableTitle}")`).click();
    cy.get(`${navSelectors.nestedNavLinks}:contains("${itemText}")`).click();
}
