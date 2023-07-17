import navSelectors from '../selectors/navigation';
import { visitMainDashboard } from './main';
import { interactAndWaitForResponses } from './request';

/**
 * For example, visitFromLeftNav('Violations');
 *
 * @param {string} itemText
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMap]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitFromLeftNav(itemText, routeMatcherMap, staticResponseMap) {
    visitMainDashboard();

    interactAndWaitForResponses(
        () => {
            cy.get(`${navSelectors.navLinks}:contains("${itemText}")`).click();
        },
        routeMatcherMap,
        staticResponseMap
    );
}

/**
 * For example, visitFromLeftNavExpandable('Vulnerability Management', 'Reporting');
 * For example, visitFromLeftNavExpandable('Platform Configuration', 'Integrations');
 *
 * @param {string} expandableTitle
 * @param {string} itemText
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMap]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitFromLeftNavExpandable(
    expandableTitle,
    itemText,
    routeMatcherMap,
    staticResponseMap
) {
    visitMainDashboard();

    interactAndWaitForResponses(
        () => {
            cy.get(`${navSelectors.navExpandable}:contains("${expandableTitle}")`).click();
            cy.get(
                `${navSelectors.navExpandable}:contains("${expandableTitle}") + ${navSelectors.nestedNavLinks}:contains("${itemText}")`
            ).click();
        },
        routeMatcherMap,
        staticResponseMap
    );
}
