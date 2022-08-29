import * as api from '../constants/apiEndpoints';

import { interceptRequests, waitForResponses } from './request';

// generic requests to render the MainPage component
const requestConfigGeneric = {
    routeMatcherMap: {
        featureflags: {
            method: 'GET',
            url: api.featureFlags,
        }, // reducers/featureFlags and sagas/featureFlagSagas
        mypermissions: {
            method: 'GET',
            url: api.roles.mypermissions,
        }, // hooks/usePermissions and reducers/roles and sagas/authSagas
        'config/public': {
            method: 'GET',
            url: api.system.configPublic,
        }, // reducers/systemConfig and sagas/systemConfig
        'auth/status': {
            method: 'GET',
            url: api.auth.authStatus,
        }, // sagas/authSagas
        credentialexpiry_CENTRAL: {
            method: 'GET',
            url: api.certExpiry.central,
        }, // MainPage/CredentialExpiryService
        credentialexpiry_SCANNER: {
            method: 'GET',
            url: api.certExpiry.scanner,
        }, // MainPage/CredentialExpiryService
    },
};

/*
 * Wait for prerequisite requests to render container components.
 *
 * Always wait on generic requests for MainPage component.
 *
 * Optionally intercept specific requests for container component:
 * routeMatcherMap: { key: routeMatcher, … }
 *
 * Optionally replace responses with stub for routeMatcher alias key:
 * staticResponseMap: { alias: { body }, … }
 * staticResponseMap: { alias: { fixture }, … }
 *
 * Optionally assign aliases for multiple GraphQL requests with routeMatcher opname key:
 * graphqlMultiAliasMap: { opname: { aliases, routeHandler }, … }
 *
 * Optionally wait for responses with waitOptions: { requestTimeout, responseTimeout }
 *
 * @param {string} pageUrl
 * @param {{ routeMatcherMap?: Record<string, { method: string, url: string }>, opnameAliasesMap?: Record<string, (request: Object) => boolean>, waitOptions?: { requestTimeout?: number, responseTimeout?: number } }} [requestConfig]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visit(pageUrl, requestConfig, staticResponseMap) {
    interceptRequests(requestConfigGeneric);
    interceptRequests(requestConfig, staticResponseMap);

    cy.visit(pageUrl);

    waitForResponses(requestConfigGeneric);
    waitForResponses(requestConfig);
}

/*
 * Visit page to test conditional rendering for user role permissions specified as response or fixture.
 *
 * { body: { resourceToAccess: { … } } }
 * { fixture: 'fixtures/wherever/whatever.json' }
 *
 * @param {string} pageUrl
 * @param {{ body: { resourceToAccess: Record<string, string> } } | { fixture: string }} permissionsStaticResponseMap
 * @param {{ routeMatcherMap?: Record<string, { method: string, url: string }>, opnameAliasesMap?: Record<string, (request: Object) => boolean>, waitOptions?: { requestTimeout?: number, responseTimeout?: number } }} [requestConfig]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitWithPermissions(
    pageUrl,
    permissionsStaticResponse,
    requestConfig,
    staticResponseMap
) {
    const staticResponseMapGeneric = {
        mypermissions: permissionsStaticResponse,
    };
    interceptRequests(requestConfigGeneric, staticResponseMapGeneric);
    interceptRequests(requestConfig, staticResponseMap);

    cy.visit(pageUrl);

    waitForResponses(requestConfigGeneric);
    waitForResponses(requestConfig);
}
