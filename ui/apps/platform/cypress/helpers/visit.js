import * as api from '../constants/apiEndpoints';

import { interceptRequests, waitForResponses } from './request';

// Import one or more alias constants in test files that call visitWithResponseMapGeneric function.
export const featureflagsAlias = 'featureflags';
export const loginAuthProviders = 'login/authproviders';
export const mypermissionsAlias = 'mypermissions';
export const configPublicAlias = 'config/public';
export const authStatusAlias = 'auth/status';
export const credentialexpiryCentralAlias = 'credentialexpiry_CENTRAL';
export const credentialexpiryScannerAlias = 'credentialexpiry_SCANNER';

// Generic requests to render the MainPage component (that is, prerequisite to test any page).
const requestConfigGeneric = {
    routeMatcherMap: {
        [featureflagsAlias]: {
            method: 'GET',
            url: api.featureFlags,
        }, // reducers/featureFlags and sagas/featureFlagSagas
        [loginAuthProviders]: {
            method: 'GET',
            url: api.auth.loginAuthProviders,
        }, // reducers/auth and sagas/authSagas
        [mypermissionsAlias]: {
            method: 'GET',
            url: api.roles.mypermissions,
        }, // hooks/usePermissions and reducers/roles and sagas/authSagas
        [configPublicAlias]: {
            method: 'GET',
            url: api.system.configPublic,
        }, // reducers/systemConfig and sagas/systemConfig
        [authStatusAlias]: {
            method: 'GET',
            url: api.auth.authStatus,
        }, // sagas/authSagas
        [credentialexpiryCentralAlias]: {
            method: 'GET',
            url: api.certExpiry.central,
        }, // MainPage/CredentialExpiryService
        [credentialexpiryScannerAlias]: {
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
 * Optional third and fouth arguments are for page-specific requests.
 *
 * { body: { resourceToAccess: { … } } }
 * { fixture: 'fixtures/wherever/whatever.json' }
 *
 * @param {string} pageUrl
 * @param {{ body: { resourceToAccess: Record<string, string> } } | { fixture: string }} staticResponseForPermissions
 * @param {{ routeMatcherMap?: Record<string, { method: string, url: string }>, opnameAliasesMap?: Record<string, (request: Object) => boolean>, waitOptions?: { requestTimeout?: number, responseTimeout?: number } }} [requestConfigSpecific]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMapSpecific]
 */
export function visitWithPermissionsResponse(
    pageUrl,
    staticResponseForPermissions,
    requestConfigSpecific,
    staticResponseMapSpecific
) {
    const staticResponseMapGeneric = {
        mypermissions: staticResponseForPermissions,
    };
    visitWithGenericResponses(
        pageUrl,
        staticResponseMapGeneric,
        requestConfigSpecific,
        staticResponseMapSpecific
    );
}

/*
 * Visit page to test one or more generic responses (for example, auth/status or credentialexpiry, possibly with mypermissions).
 * Optional third and fouth arguments are for page-specific requests.
 *
 * Examples
 * { [authStatusAlias]: { body: {}, statusCode: 401 } }
 * { [credentialExpiryCentralAlias]: { expiry } }
 *
 * @param {string} pageUrl
 * @param {{ body: unknown } | { fixture: string }} staticResponseMapGeneric
 * @param {{ routeMatcherMap?: Record<string, { method: string, url: string }>, opnameAliasesMap?: Record<string, (request: Object) => boolean>, waitOptions?: { requestTimeout?: number, responseTimeout?: number } }} [requestConfigSpecific]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMapSpecific]
 */
export function visitWithGenericResponses(
    pageUrl,
    staticResponseMapGeneric,
    requestConfigSpecific,
    staticResponseMapSpecific
) {
    interceptRequests(requestConfigGeneric, staticResponseMapGeneric);
    interceptRequests(requestConfigSpecific, staticResponseMapSpecific);

    cy.visit(pageUrl);

    waitForResponses(requestConfigGeneric);
    waitForResponses(requestConfigSpecific);
}
