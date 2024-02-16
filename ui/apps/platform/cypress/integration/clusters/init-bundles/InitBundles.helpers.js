import { interactAndWaitForResponses } from '../../../helpers/request';
import { visit } from '../../../helpers/visit';

const pagePath = '/main/clusters/init-bundles';
const formAction = '?action=create';

// routeMatcherMap

const urlForInitBundles = '/v1/cluster-init/init-bundles';

export const initBundlesAlias = 'init-bundles';

const routeMatcherMapForInitBundles = {
    [initBundlesAlias]: {
        method: 'GET',
        url: urlForInitBundles,
    },
};

// assert

export function assertInitBundleForm() {
    cy.location('pathname').should('eq', pagePath);
    cy.location('search').should('eq', formAction);
    cy.get('h1:contains("Create bundle")');
}

export function assertInitBundlePage() {
    cy.location('pathname').should('contain', pagePath); // contain because of id
    cy.contains('h1', /^Cluster init bundle$/); // singular
}

export function assertInitBundlesPage() {
    cy.location('pathname').should('eq', pagePath);
    cy.contains('h1', /^Cluster init bundles$/); // plural
}

// interact

/**
 * Visit init bundles or init bundle page.
 *
 * @param {function} interactionCallback
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function interactAndWaitForInitBundles(interactionCallback, staticResponseMap) {
    return interactAndWaitForResponses(
        () => {
            interactionCallback();
        },
        routeMatcherMapForInitBundles,
        staticResponseMap
    );
}

/**
 * Create bundle and go back to init bundles page.
 *
 * @param {function} interactionCallback
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function interactAndWaitForCreateBundle(interactionCallback, staticResponseMap) {
    return interactAndWaitForResponses(
        () => {
            interactionCallback();
        },
        {
            'POST_init-bundles': {
                method: 'POST',
                url: urlForInitBundles,
            },
            ...routeMatcherMapForInitBundles,
        },
        staticResponseMap
    );
}

/**
 * Revoke bundle via either button in page or row action in table.
 *
 * @param {function} interactionCallback
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function interactAndWaitForRevokeBundle(interactionCallback, staticResponseMap) {
    return interactAndWaitForResponses(
        () => {
            interactionCallback();
        },
        {
            'PATCH_init-bundles': {
                method: 'PATCH',
                url: `${urlForInitBundles}/revoke`,
            },
            ...routeMatcherMapForInitBundles,
        },
        staticResponseMap
    );
}

// visit

export function visitInitBundleForm() {
    visit(`${pagePath}${formAction}`);

    assertInitBundleForm();
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitInitBundlesPage(staticResponseMap) {
    visit(pagePath, routeMatcherMapForInitBundles, staticResponseMap);

    assertInitBundlesPage();
}
