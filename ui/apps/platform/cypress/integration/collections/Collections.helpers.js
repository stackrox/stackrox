import navSelectors from '../../selectors/navigation';

import { visitFromLeftNavExpandable } from '../../helpers/nav';
import { visit } from '../../helpers/visit';
import { collectionSelectors } from './Collections.selectors';

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

const baseApiUrl = '/v1/collections';

export function tryCreateCollection(
    name,
    description,
    embeddedCollectionIds = [],
    resourceSelectors = []
) {
    const auth = { bearer: Cypress.env('ROX_AUTH_TOKEN') };

    cy.request({
        url: `${baseApiUrl}?query.query=Collection Name:"${name}"`,
        auth,
    }).as('listCollections');

    cy.get('@listCollections').then((res) => {
        if (res.body.collections.some((c) => c.name === name)) {
            // Collection already exists
            return;
        }
        const body = {
            name,
            description,
            embeddedCollectionIds,
            resourceSelectors,
        };
        cy.request({ url: baseApiUrl, body, auth, method: 'POST' });
    });
}

// Cleanup an existing collection via API call
export function tryDeleteCollection(collectionName) {
    const auth = { bearer: Cypress.env('ROX_AUTH_TOKEN') };

    cy.request({
        url: `${baseApiUrl}?query.query=Collection Name:"${collectionName}"`,
        auth,
    }).as('listCollections');

    cy.get('@listCollections').then((res) => {
        const collection = res.body.collections.find(({ name }) => name === collectionName);
        if (collection) {
            const { id } = collection;
            const url = `${baseApiUrl}/${id}`;
            cy.request({ url, auth, method: 'DELETE' });
        }
    });
}

/**
 * This ensures that the deployments passed as an argument are matched in the results panel, and that
 * no other deployments are matched. Note that this is limited to less than 10 deployments, as we
 * also assert that the "View more" button does not exist. Otherwise it is possible deployments that
 * do not exist in the passed argument array are in fact matched, leading to an incorrect assertion.
 * @param {string[]} deployments
 */
export function assertDeploymentsAreMatchedExactly(deployments) {
    assertDeploymentsAreMatched(deployments);
    cy.get(collectionSelectors.deploymentResults).its('length').should('be.eq', deployments.length);
    cy.get(collectionSelectors.viewMoreResultsButton).should('not.exist');
}

export function assertDeploymentsAreMatched(deployments) {
    deployments.forEach((deployment) => cy.get(collectionSelectors.deploymentResult(deployment)));
}

export function assertDeploymentsAreNotMatched(deployments) {
    deployments.forEach((deployment) =>
        cy.get(collectionSelectors.deploymentResult(deployment)).should('not.exist')
    );
}
