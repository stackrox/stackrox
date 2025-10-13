import { interactAndWaitForResponses } from '../../../helpers/request';
import { visit } from '../../../helpers/visit';

const pagePath = '/main/clusters/discovered-clusters';

// route

export const countDiscoveredClustersAlias = 'count_discovered-clusters';
export const discoveredClustersAlias = 'discovered-clusters';
export const cloudSourcesAlias = 'cloud-sources';

const routeMatcherMapForTable = {
    [countDiscoveredClustersAlias]: {
        method: 'GET',
        url: '/v1/count/discovered-clusters?*',
    },
    [discoveredClustersAlias]: {
        method: 'GET',
        url: '/v1/discovered-clusters?*',
    },
};

const routeMatcherMapForPage = {
    ...routeMatcherMapForTable,
    [cloudSourcesAlias]: {
        method: 'GET',
        url: '/v1/cloud-sources',
    },
};

// assert

export function assertDiscoveredClustersPage() {
    cy.location('pathname').should('eq', pagePath);
    cy.get('h1:contains("Discovered clusters")');
}

export function assertSortByColumn(text, direction, search) {
    cy.get(`th:contains("${text}")[aria-sort="${direction}"]`);
    cy.location('search').should('eq', search);
}

// interact

export function sortByColumn(text) {
    return interactAndWaitForResponses(() => {
        cy.get(`th:contains("${text}")`).click();
    }, routeMatcherMapForTable);
}

// visit

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitDiscoveredClusters(staticResponseMap) {
    visit(pagePath, routeMatcherMapForPage, staticResponseMap);
}
