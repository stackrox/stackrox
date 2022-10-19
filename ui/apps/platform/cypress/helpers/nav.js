import navSelectors from '../selectors/navigation';
import { visitMainDashboard } from './main';
import { interactAndWaitForResponses } from './request';

/*
 * For example, visitFromLeftNav('Violations');
 */
export function visitFromLeftNav(itemText, requestConfig, staticResponseMap) {
    visitMainDashboard();

    interactAndWaitForResponses(
        () => {
            cy.get(`${navSelectors.navLinks}:contains("${itemText}")`).click();
        },
        requestConfig,
        staticResponseMap
    );
}

/*
 * For example, visitFromLeftNavExpandable('Vulnerability Management', 'Reporting');
 * For example, visitFromLeftNavExpandable('Platform Configuration', 'Integrations');
 */
export function visitFromLeftNavExpandable(
    expandableTitle,
    itemText,
    requestConfig,
    staticResponseMap
) {
    visitMainDashboard();

    interactAndWaitForResponses(
        () => {
            cy.get(`${navSelectors.navExpandable}:contains("${expandableTitle}")`).click();
            cy.get(`${navSelectors.nestedNavLinks}:contains("${itemText}")`).click();
        },
        requestConfig,
        staticResponseMap
    );
}
