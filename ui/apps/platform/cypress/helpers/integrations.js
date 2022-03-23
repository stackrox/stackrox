/* eslint-disable import/prefer-default-export */
import * as api from '../constants/apiEndpoints';

export function visitIntegrationsUrl(url) {
    cy.intercept('GET', api.integrations.apiTokens).as('getAPITokens');
    cy.intercept('GET', api.integrations.clusterInitBundles).as('getClusterInitBundles');
    cy.intercept('GET', api.integrations.externalBackups).as('getBackupIntegrations');
    cy.intercept('GET', api.integrations.imageIntegrations).as('getImageIntegrations');
    cy.intercept('GET', api.integrations.notifiers).as('getNotifierIntegrations');
    // TODO: add signature integrations after ROX_VERIFY_IMAGE_SIGNATURE is enabled by default

    cy.visit(url);

    cy.wait([
        '@getAPITokens',
        '@getClusterInitBundles',
        '@getBackupIntegrations',
        '@getImageIntegrations',
        '@getNotifierIntegrations',
    ]);
}
