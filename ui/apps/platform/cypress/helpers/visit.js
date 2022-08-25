import * as api from '../constants/apiEndpoints';

import { interceptRequests, waitForResponses } from './request';

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
    cy.intercept('GET', api.featureFlags).as('featureflags');
    cy.intercept('GET', api.roles.mypermissions).as('mypermissions');
    cy.intercept('GET', api.system.configPublic).as('config/public');
    interceptRequests(requestConfig, staticResponseMap);

    cy.visit(pageUrl);

    cy.wait(['@featureflags', '@mypermissions', '@config/public']);
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
    cy.intercept('GET', api.featureFlags).as('featureflags');
    cy.intercept('GET', api.roles.mypermissions, permissionsStaticResponse).as('mypermissions');
    cy.intercept('GET', api.system.configPublic).as('config/public');
    interceptRequests(requestConfig, staticResponseMap);

    cy.visit(pageUrl);

    cy.wait(['@featureflags', '@mypermissions', '@config/public']);
    waitForResponses(requestConfig);
}
