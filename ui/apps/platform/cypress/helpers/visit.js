import { interceptRequests, waitForResponses } from './request';

// Single source of truth for keys in staticResponseMapForAuthenticatedRoutes object.
export const availableAuthProvidersAlias = 'availableAuthProviders';
export const featureFlagsAlias = 'featureflags';
export const loginAuthProvidersAlias = 'login/authproviders';
export const myPermissionsAlias = 'mypermissions';
export const configPublicAlias = 'config/public';
export const authStatusAlias = 'auth/status';
export const centralCapabilitiesAlias = 'central-capabilities';

// Requests to render pages via MainPage and Body components.
const routeMatcherMapForAuthenticatedRoutes = {
    [availableAuthProvidersAlias]: {
        method: 'GET',
        url: '/v1/availableAuthProviders',
    }, // reducers/auth and sagas/authSagas
    [featureFlagsAlias]: {
        method: 'GET',
        url: '/v1/featureflags',
    }, // reducers/featureFlags and sagas/featureFlagSagas
    [loginAuthProvidersAlias]: {
        method: 'GET',
        url: '/v1/login/authproviders',
    }, // reducers/auth and sagas/authSagas
    [myPermissionsAlias]: {
        method: 'GET',
        url: 'v1/mypermissions',
    }, // hooks/usePermissions and reducers/roles and sagas/authSagas
    [configPublicAlias]: {
        method: 'GET',
        url: '/v1/config/public',
    }, // reducers/systemConfig and sagas/systemConfig
    [authStatusAlias]: {
        method: 'GET',
        url: '/v1/auth/status',
    }, // sagas/authSagas,
    [centralCapabilitiesAlias]: {
        method: 'GET',
        url: '/v1/central-capabilities',
    }, // reducers/centralCapabilities,
    /*
     * Intentionally omit credentialexpiry requests for central and scanner,
     * because they are in parallel with (and possibly even delayed by) page-specific requests.
     */
};

/**
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
 * @returns {{ request: Record<string, unknown>, response: Record<string, unknown>}[]}
 */
export function visit(pageUrl, routeMatcherMap, staticResponseMap) {
    interceptRequests(routeMatcherMapForAuthenticatedRoutes);
    interceptRequests(routeMatcherMap, staticResponseMap);

    cy.visit(pageUrl);

    waitForResponses(routeMatcherMapForAuthenticatedRoutes);
    return waitForResponses(routeMatcherMap);
}

/**
 * Visit page to test conditional rendering for authentication status specified as response or fixture.
 *
 * { body: { resourceToAccess: { … } } }
 * { fixture: 'fixtures/wherever/whatever.json' }
 *
 * @param {string} pageUrl
 * @param {{ body: { userInfo: Record<string, unknown> } } | { fixture: string }} staticResponseForAuthStatus
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMap]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitWithStaticResponseForAuthStatus(
    pageUrl,
    staticResponseForAuthStatus,
    routeMatcherMap,
    staticResponseMap
) {
    const staticResponseMapForAuthenticatedRoutes = {
        [authStatusAlias]: staticResponseForAuthStatus,
    };
    interceptRequests(
        routeMatcherMapForAuthenticatedRoutes,
        staticResponseMapForAuthenticatedRoutes
    );
    interceptRequests(routeMatcherMap, staticResponseMap);

    cy.visit(pageUrl);

    waitForResponses(routeMatcherMapForAuthenticatedRoutes);
    waitForResponses(routeMatcherMap);
}

/**
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
    const staticResponseMapForAuthenticatedRoutes = {
        [myPermissionsAlias]: staticResponseForPermissions,
    };
    interceptRequests(
        routeMatcherMapForAuthenticatedRoutes,
        staticResponseMapForAuthenticatedRoutes
    );
    interceptRequests(routeMatcherMap, staticResponseMap);

    cy.visit(pageUrl);

    waitForResponses(routeMatcherMapForAuthenticatedRoutes);
    waitForResponses(routeMatcherMap);
}

/**
 * Visit page to test conditional rendering for central capabilities specified as response or fixture.
 *
 * { body: { ... } }
 * { fixture: 'fixtures/wherever/whatever.json' }
 *
 * @param {string} pageUrl
 * @param {{ body: { [key: string]: 'CapabilityAvailable' | 'CapabilityDisabled' } }} staticResponseForCapabilities
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMap]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitWithStaticResponseForCapabilities(
    pageUrl,
    staticResponseForCapabilities,
    routeMatcherMap,
    staticResponseMap
) {
    const staticResponseMapForAuthenticatedRoutes = {
        [centralCapabilitiesAlias]: staticResponseForCapabilities,
    };
    interceptRequests(
        routeMatcherMapForAuthenticatedRoutes,
        staticResponseMapForAuthenticatedRoutes
    );
    interceptRequests(routeMatcherMap, staticResponseMap);

    cy.visit(pageUrl);

    waitForResponses(routeMatcherMapForAuthenticatedRoutes);
    waitForResponses(routeMatcherMap);
}

export function assertCannotFindThePage() {
    cy.get('h1:contains("Cannot find the page")');
}
