import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import {
    clickCreateNewIntegrationInTable,
    clickIntegrationSourceLinkInForm,
    deleteIntegrationInTable,
    saveCreatedIntegrationInForm,
    testIntegrationInFormWithoutStoredCredentials,
    visitIntegrationsTable,
} from './integrations.helpers';
import { selectors } from './integrations.selectors';
import {
    generateNameWithDate,
    getHelperElementByLabel,
    getInputByLabel,
} from '../../helpers/formHelpers';

const integrationSource = 'aiIntegrations';
const integrationType = 'lightspeed';

describe('AI Integrations - OpenShift Lightspeed', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_AI_INTEGRATIONS')) {
            this.skip();
        }
    });

    it('should create a new Lightspeed integration, test it, save, view, and delete', () => {
        const integrationName = generateNameWithDate('Lightspeed Integration');

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        // Check initial state - Save button should be disabled
        cy.get(selectors.buttons.save).should('be.disabled');

        // Check empty name is not accepted
        getInputByLabel('Integration name').type(' ').blur();
        getHelperElementByLabel('Integration name').contains('Integration name is required');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Fill in valid integration details
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Service URL').clear().type('https://lightspeed.example.com');

        // Test button should be enabled with valid data
        cy.get(selectors.buttons.test).should('be.enabled');

        // Test the integration (mock success response)
        const staticResponseForTest = { body: {} };
        testIntegrationInFormWithoutStoredCredentials(
            integrationSource,
            integrationType,
            staticResponseForTest
        );

        // Verify test success message appears
        cy.get('.pf-v6-c-alert.pf-m-success').should('be.visible');
        cy.get('.pf-v6-c-alert.pf-m-success').contains('The test was successful');

        // Save button should be enabled after filling required fields
        cy.get(selectors.buttons.save).should('be.enabled');

        // Save the integration
        saveCreatedIntegrationInForm(integrationSource, integrationType);

        // View the integration
        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).click();
        cy.get(`${selectors.breadcrumbItem}:contains("${integrationName}")`);

        // Go back to list
        clickIntegrationSourceLinkInForm(integrationSource, integrationType);

        // Delete the integration
        deleteIntegrationInTable(integrationSource, integrationType, integrationName);

        // Verify it's gone
        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).should('not.exist');
    });

    it('should show validation error when trying to save without required fields', () => {
        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        // Try to interact with save button without filling name
        getInputByLabel('Integration name').focus().blur();

        cy.get(selectors.buttons.save).should('be.disabled');

        // Fill only service URL (optional field)
        getInputByLabel('Service URL').type('https://lightspeed.example.com');

        // Save should still be disabled because name is required
        cy.get(selectors.buttons.save).should('be.disabled');

        // Cancel to clean up
        cy.get(selectors.buttons.cancel).click();
    });

    it('should reject creating a second integration when one already exists (singleton enforcement)', () => {
        const firstIntegrationName = generateNameWithDate('First Lightspeed');
        const secondIntegrationName = generateNameWithDate('Second Lightspeed');

        // Create first integration
        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        getInputByLabel('Integration name').clear().type(firstIntegrationName);
        getInputByLabel('Service URL').clear().type('https://lightspeed-1.example.com');

        saveCreatedIntegrationInForm(integrationSource, integrationType);

        // Verify it was created
        cy.get(`${selectors.tableRowNameLink}:contains("${firstIntegrationName}")`).should('exist');

        // Try to create a second integration
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        getInputByLabel('Integration name').clear().type(secondIntegrationName);
        getInputByLabel('Service URL').clear().type('https://lightspeed-2.example.com');

        // Mock the singleton rejection error from backend
        const errorResponse = {
            statusCode: 400,
            body: {
                error: 'Only one OpenShift Lightspeed integration is allowed',
                message: 'Only one OpenShift Lightspeed integration is allowed',
            },
        };

        cy.intercept('POST', '/v2/ai-integrations', errorResponse).as('createRejected');

        cy.get(selectors.buttons.save).click();

        cy.wait('@createRejected');

        // Verify error banner appears
        cy.get('.pf-v6-c-alert.pf-m-danger').should('be.visible');
        cy.get('.pf-v6-c-alert.pf-m-danger').contains('Only one');

        // Cancel and clean up the first integration
        cy.get(selectors.buttons.cancel).click();

        deleteIntegrationInTable(integrationSource, integrationType, firstIntegrationName);
        cy.get(`${selectors.tableRowNameLink}:contains("${firstIntegrationName}")`).should(
            'not.exist'
        );
    });
});
