import navSelectors from '../selectors/navigation';

import { visitFromLeftNavExpandable } from './nav';
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

// visit

export function visitCollections(staticResponseMap) {
    visit(basePath, routeMatcherMapForCollections, staticResponseMap);

    cy.get('h1:contains("Collections")');
    cy.get(`${navSelectors.navExpandable}:contains("Platform Configuration")`);
    cy.get(`${navSelectors.nestedNavLinks}:contains("Collections")`).should(
        'have.class',
        'pf-m-current'
    );
}

export function visitCollectionsFromLeftNav(staticResponseMap) {
    visitFromLeftNavExpandable(
        'Platform Configuration',
        'Collections',
        routeMatcherMapForCollections,
        staticResponseMap
    );

    cy.get('h1:contains("Collections")');
    cy.location('pathname').should('eq', basePath);
}
