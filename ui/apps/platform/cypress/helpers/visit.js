import * as api from '../constants/apiEndpoints';

import { interceptRequests, waitForResponses } from './request';

// Single source of truth for keys in optional staticResponseMap argument.
export const availableAuthProvidersAlias = 'availableAuthProviders';
export const featureFlagsAlias = 'featureflags';
export const loginAuthProvidersAlias = 'login/authproviders';
export const myPermissionsAlias = 'mypermissions';
export const configPublicAlias = 'config/public';
export const authStatusAlias = 'auth/status';

// generic requests to render the MainPage component
const routeMatcherMapGeneric = {
    [availableAuthProvidersAlias]: {
        method: 'GET',
        url: api.auth.availableAuthProviders,
    }, // reducers/auth and sagas/authSagas
    [featureFlagsAlias]: {
        method: 'GET',
        url: api.featureFlags,
    }, // reducers/featureFlags and sagas/featureFlagSagas
    [loginAuthProvidersAlias]: {
        method: 'GET',
        url: api.auth.loginAuthProviders,
    }, // reducers/auth and sagas/authSagas
    [myPermissionsAlias]: {
        method: 'GET',
        url: api.roles.mypermissions,
    }, // hooks/usePermissions and reducers/roles and sagas/authSagas
    [configPublicAlias]: {
        method: 'GET',
        url: '/v1/config/public',
    }, // reducers/systemConfig and sagas/systemConfig
    [authStatusAlias]: {
        method: 'GET',
        url: api.auth.authStatus,
    }, // sagas/authSagas
    /*
     * Intentionally omit credentialexpiry requests for central and scanner,
     * because they are in parallel with (and possibly even delayed by) page-specific requests.
     */
};

/*
 * Wait for prerequisite requests to render container components.
 *
 * Always wait on generic requests for MainPage component.
 *
 * Optionally intercept specific requests for container component:
 * routeMatcherMap: { alias: routeMatcher, … }
 *
 * Optionally replace responses with stub for routeMatcher alias key:
 * staticResponseMap: { alias: { body }, … }
 * staticResponseMap: { alias: { fixture }, … }
 *
 * @param {string} pageUrl
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMap]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visit(pageUrl, routeMatcherMap, staticResponseMap) {
    interceptRequests(routeMatcherMapGeneric);
    interceptRequests(routeMatcherMap, staticResponseMap);

    cy.visit(pageUrl);

    waitForResponses(routeMatcherMapGeneric);
    waitForResponses(routeMatcherMap);
}

/*
 * Visit page to test conditional rendering for user role permissions specified as response or fixture.
 *
 * { body: { resourceToAccess: { … } } }
 * { fixture: 'fixtures/wherever/whatever.json' }
 *
 * @param {string} pageUrl
 * @param {{ body: { resourceToAccess: Record<string, string> } } | { fixture: string }} staticResponseForPermissions
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMap]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitWithStaticResponseForPermissions(
    pageUrl,
    staticResponseForPermissions,
    routeMatcherMap,
    staticResponseMap
) {
    const staticResponseMapGeneric = {
        [myPermissionsAlias]: staticResponseForPermissions,
    };
    interceptRequests(routeMatcherMapGeneric, staticResponseMapGeneric);
    interceptRequests(routeMatcherMap, staticResponseMap);

    cy.visit(pageUrl);

    waitForResponses(routeMatcherMapGeneric);
    waitForResponses(routeMatcherMap);
}
