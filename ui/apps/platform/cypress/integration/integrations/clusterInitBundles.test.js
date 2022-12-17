import withAuth from '../../helpers/basicAuth';
import {
    getHelperElementByLabel,
    getInputByLabel,
    generateNameWithRandomString,
} from '../../helpers/formHelpers';

import {
    assertIntegrationsTable,
    clickCreateNewIntegrationInTable,
    clickIntegrationSourceLinkInForm,
    generateCreatedAuthProvidersIntegrationInForm,
    revokeAuthProvidersIntegrationInTable,
    visitIntegrationsTable,
} from './integrations.helpers';
import { selectors } from './integrations.selectors';

// Page address segments are the source of truth for integrationSource and integrationType.
const integrationSource = 'authProviders';
const integrationType = 'clusterInitBundle';

describe('Cluster Init Bundles', () => {
    withAuth();

    it('should create a new Cluster Init Bundle and then view and delete', () => {
        // we have to use a randomstring here because using a name with a date is not a valid clusterInitBundle name
        const clusterInitBundleName = generateNameWithRandomString('ClusterInitBundleTest');

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType, 'Generate bundle');

        // Step 0, should start out with disabled Generate button
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Cluster init bundle name').type(' ').blur();

        getHelperElementByLabel('Cluster init bundle name').contains(
            'A cluster init bundle name is required'
        );
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 2, check fields for invalid formats
        getInputByLabel('Cluster init bundle name').type('Name with space/stuff').blur();

        getHelperElementByLabel('Cluster init bundle name').contains(
            'Name must contain only alphanumeric, ., _, or - (no spaces).'
        );
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 3, check valid from and generate
        getInputByLabel('Cluster init bundle name').clear().type(clusterInitBundleName);

        generateCreatedAuthProvidersIntegrationInForm(integrationType);

        cy.get('[aria-label="Success Alert"]').should('contain', 'Download Helm values file');
        cy.get('[aria-label="Success Alert"]').should(
            'contain',
            'Download Kubernetes secrets file'
        );

        cy.get(selectors.buttons.back).click();

        // View it.

        assertIntegrationsTable(integrationSource, integrationType);

        cy.get(`${selectors.tableRowNameLink}:contains("${clusterInitBundleName}")`).click();

        cy.get(`${selectors.breadcrumbItem}:contains("${clusterInitBundleName}")`);

        clickIntegrationSourceLinkInForm(integrationSource, integrationType);

        // Revoke it.

        revokeAuthProvidersIntegrationInTable(integrationType, clusterInitBundleName);

        cy.get(`${selectors.tableRowNameLink}:contains("${clusterInitBundleName}")`).should(
            'not.exist'
        );
    });
});
