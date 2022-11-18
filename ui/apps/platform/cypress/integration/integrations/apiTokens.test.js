import { selectors } from '../../constants/IntegrationsPage';
import withAuth from '../../helpers/basicAuth';
import {
    getHelperElementByLabel,
    getInputByLabel,
    generateNameWithDate,
    getSelectButtonByLabel,
    getSelectOption,
} from '../../helpers/formHelpers';
import {
    assertIntegrationsTable,
    clickCreateNewIntegrationInTable,
    revokeAuthProvidersIntegrationInTable,
    saveCreatedIntegrationInForm,
    visitIntegrationsTable,
} from '../../helpers/integrations';

// Page address segments are the source of truth for integrationSource and integrationType.
const integrationSource = 'authproviders';
const integrationType = 'apitoken';

describe('API Tokens', () => {
    withAuth();

    it('should create a new API Token and then view and delete', () => {
        const apiTokenName = generateNameWithDate('API Token Test');

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType, 'Generate token');

        // Step 0, should start out with disabled Generate button
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Token name').type(' ').blur();

        getHelperElementByLabel('Token name').contains('A token name is required');
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 2, check valid from and generate
        getInputByLabel('Token name').clear().type(apiTokenName);
        getSelectButtonByLabel('Role').click();
        getSelectOption('Admin').click();

        saveCreatedIntegrationInForm(integrationSource, integrationType);

        cy.get('[aria-label="Success Alert"]');

        cy.get(selectors.buttons.back).click();

        // View it.

        assertIntegrationsTable(integrationSource, integrationType);

        cy.get(`${selectors.tableRowNameLink}:contains("${apiTokenName}")`).click();

        cy.get(`${selectors.breadcrumbItem}:contains("${apiTokenName}")`);

        // Revoke it.

        revokeAuthProvidersIntegrationInTable(integrationType, apiTokenName);

        cy.get(`${selectors.tableRowNameLink}:contains("${apiTokenName}")`).should('not.exist');
    });
});
