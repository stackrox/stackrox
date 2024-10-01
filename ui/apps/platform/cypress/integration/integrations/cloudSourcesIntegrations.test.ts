import withAuth from '../../helpers/basicAuth';
import {
    clickCreateNewIntegrationInTable,
    clickIntegrationSourceLinkInForm,
    deleteIntegrationInTable,
    saveCreatedIntegrationInForm,
    visitIntegrationsTable,
} from './integrations.helpers';
import { selectors } from './integrations.selectors';
import {
    generateNameWithDate,
    getHelperElementByLabel,
    getInputByLabel,
} from '../../helpers/formHelpers';

const integrationSource = 'cloudSources';

describe('Cloud Source Integrations', () => {
    withAuth();

    it('should create a new Paladin Cloud integration and then view and delete', () => {
        const integrationName = generateNameWithDate('Paladin Cloud Integration');
        const integrationType = 'paladinCloud';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        // Check initial state.
        cy.get(selectors.buttons.save).should('be.disabled');

        // // Check empty values are not accepted.
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Paladin Cloud endpoint').clear().type(' ');
        getInputByLabel('Paladin Cloud token').clear().type(' ').blur();

        getHelperElementByLabel('Integration name').contains('Integration name is required');
        getHelperElementByLabel('Paladin Cloud endpoint').contains('Endpoint is required');
        getHelperElementByLabel('Paladin Cloud token').contains('Token is required');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Save integration.
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Paladin Cloud endpoint').clear().type('https://stackrox.io');
        getInputByLabel('Paladin Cloud token').clear().type('tokenvalue');

        saveCreatedIntegrationInForm(integrationSource, integrationType);

        // View it.
        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).click();

        cy.get(`${selectors.breadcrumbItem}:contains("${integrationName}")`);

        clickIntegrationSourceLinkInForm(integrationSource, integrationType);

        // Delete it.
        deleteIntegrationInTable(integrationSource, integrationType, integrationName);

        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).should('not.exist');
    });

    it('should create a new OCM integration and then view and delete', () => {
        const integrationName = generateNameWithDate('OCM Integration');
        const integrationType = 'ocm';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        // Check initial state.
        cy.get(selectors.buttons.save).should('be.disabled');

        // // Check empty values are not accepted.
        getInputByLabel('Integration name').type(' ');
        getInputByLabel('Endpoint').clear().type(' ');
        getInputByLabel('Client ID').clear().type(' ');
        getInputByLabel('Client secret').clear().type(' ').blur();
        getInputByLabel('API token').clear().type(' ').blur();

        getHelperElementByLabel('Integration name').contains('Integration name is required');
        getHelperElementByLabel('Endpoint').contains('Endpoint is required');
        getHelperElementByLabel('Client ID').contains('Client ID is required');
        getHelperElementByLabel('Client secret').contains('Client secret is required');
        getHelperElementByLabel('API token').contains('Token is required');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Save integration.
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Endpoint').clear().type('https://stackrox.io');

        // // Check that entering the API token enables the save button.
        getInputByLabel('API token').clear().type('api_token');
        cy.get(selectors.buttons.save).should('be.enabled');

        // // Clear API token again and save the integration with client credentials.
        getInputByLabel('API token').clear();
        getInputByLabel('Client ID').clear().type('client_id');
        getInputByLabel('Client secret').clear().type('client_secret');

        saveCreatedIntegrationInForm(integrationSource, integrationType);

        // View it.
        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).click();

        cy.get(`${selectors.breadcrumbItem}:contains("${integrationName}")`);

        clickIntegrationSourceLinkInForm(integrationSource, integrationType);

        // Delete it.
        deleteIntegrationInTable(integrationSource, integrationType, integrationName);

        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).should('not.exist');
    });
});
