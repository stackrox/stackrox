/* eslint-disable import/prefer-default-export */
import * as api from '../constants/apiEndpoints';

export function visitIntegrationsUrl(url) {
    cy.intercept('GET', api.integrations.apiTokens).as('getAPITokens');
    cy.intercept('GET', api.integrations.clusterInitBundles).as('getClusterInitBundles');
    cy.intercept('GET', api.integrations.externalBackups).as('getBackupIntegrations');
    cy.intercept('GET', api.integrations.imageIntegrations).as('getImageIntegrations');
    cy.intercept('GET', api.integrations.notifiers).as('getNotifierIntegrations');

    cy.visit(url);

    cy.wait([
        '@getAPITokens',
        '@getClusterInitBundles',
        '@getBackupIntegrations',
        '@getImageIntegrations',
        '@getNotifierIntegrations',
    ]);

    /*
     * Wait so New integration button does not become detached when IntegrationsTable element rerenders.
     * Wait might become unnecessary if there is an alternative to callback function idiom:
     * components={(props) => <Link {...props} … />} for Button element.
     * Rendering via function instead of children might cause React to replace the button element.
     */
    cy.wait(100);
}
