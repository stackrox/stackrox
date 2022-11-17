import navSelectors from '../selectors/navigation';

import { visitFromLeftNavExpandable } from './nav';
import { interceptAndWaitForResponses } from './request';
import { visit } from './visit';

const basePath = '/main/collections';

export const collectionsAlias = 'collections';
export const collectionsCountAlias = 'collections/count';

const routeMatcherMapForCollections = {
    [collectionsAlias]: {
        method: 'GET',
        url: '/v1/collections?query.query=*',
    },
    [collectionsCountAlias]: {
        method: 'GET',
        url: '/v1/collectionscount?query.query=*',
    },
};

const title = 'Collections';

// visit

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitCollections(staticResponseMap) {
    visit(basePath);

    cy.get(`h1:contains("${title}")`);
    cy.get(`${navSelectors.navExpandable}:contains("Platform Configuration")`);
    cy.get(`${navSelectors.nestedNavLinks}:contains("${title}")`).should(
        'have.class',
        'pf-m-current'
    );

    interceptAndWaitForResponses(routeMatcherMapForCollections, staticResponseMap);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitCollectionsFromLeftNav(staticResponseMap) {
    visitFromLeftNavExpandable('Platform Configuration', title);

    cy.get('h1:contains("Collections")');
    cy.location('pathname').should('eq', basePath);

    interceptAndWaitForResponses(routeMatcherMapForCollections, staticResponseMap);
}
