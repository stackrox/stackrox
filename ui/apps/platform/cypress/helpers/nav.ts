import navSelectors from '../selectors/navigation';
import { visitMainDashboard } from './main';
import { interactAndWaitForResponses } from './request';

/**
 * For example, visitFromLeftNav('Violations');
 */
export function visitFromLeftNav(
    itemText: string,
    routeMatcherMap: Record<string, { method: string; url: string }>,
    staticResponseMap: Record<string, { body: unknown } | { fixture: string }>
) {
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
 */
export function visitFromLeftNavExpandable(
    expandableTitle: string,
    itemText: string,
    routeMatcherMap: Record<string, { method: string; url: string }>,
    staticResponseMap: Record<string, { body: unknown } | { fixture: string }>
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

export function visitFromHorizontalNav(linkTitle: string) {
    cy.get(`${navSelectors.horizontalNavLinks}:contains("${linkTitle}")`).click();
}

export function visitFromHorizontalNavExpandable(expandableItemTitle: string) {
    return (linkTitle: string) => {
        cy.get(`nav.pf-m-horizontal-subnav button:contains("${expandableItemTitle}")`).click();
        cy.get(
            `nav.pf-m-horizontal-subnav .pf-v5-c-menu a[role="menuitem"]:contains("${linkTitle}")`
        ).click();
    };
}
