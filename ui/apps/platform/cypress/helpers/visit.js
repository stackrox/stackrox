import * as api from '../constants/apiEndpoints';

/*
 * Wait for prerequisite requests to render container components.
 */
// eslint-disable-next-line import/prefer-default-export
export function visit(url) {
    cy.intercept('GET', api.featureFlags).as('getFeatureFlags');
    cy.intercept('GET', api.roles.mypermissions).as('getMyPermissions');
    cy.intercept('GET', api.system.configPublic).as('getConfigPublic');
    cy.visit(url);
    cy.wait(['@getFeatureFlags', '@getMyPermissions', '@getConfigPublic']);
}
