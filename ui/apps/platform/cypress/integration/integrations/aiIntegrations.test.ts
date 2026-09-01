import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import {
    aiIntegrationTestNamePrefix,
    cleanupAiIntegrations,
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

    beforeEach(() => {
        cleanupAiIntegrations();
    });

    afterEach(() => {
        cleanupAiIntegrations();
    });

    it('should create a new Lightspeed integration, test it, save, view, and delete', () => {
        const integrationName = generateNameWithDate(aiIntegrationTestNamePrefix);

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        cy.get(selectors.buttons.save).should('be.disabled');

        getInputByLabel('Integration name').type(' ').blur();
        getHelperElementByLabel('Integration name').contains('Integration name is required');
        cy.get(selectors.buttons.save).should('be.disabled');

        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Service URL').clear().type('https://lightspeed.example.com');

        cy.get(selectors.buttons.test).should('be.enabled');

        testIntegrationInFormWithoutStoredCredentials(integrationSource, integrationType);

        cy.get('.pf-v6-c-alert.pf-m-success').should('be.visible');
        cy.get('.pf-v6-c-alert.pf-m-success').contains('The test was successful');

        cy.get(selectors.buttons.save).should('be.enabled');

        saveCreatedIntegrationInForm(integrationSource, integrationType);

        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).click();
        cy.get(`${selectors.breadcrumbItem}:contains("${integrationName}")`);

        clickIntegrationSourceLinkInForm(integrationSource, integrationType);

        deleteIntegrationInTable(integrationSource, integrationType, integrationName);

        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).should('not.exist');
    });

    it('should show validation error when trying to save without required fields', () => {
        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        getInputByLabel('Integration name').focus().blur();

        cy.get(selectors.buttons.save).should('be.disabled');

        getInputByLabel('Service URL').type('https://lightspeed.example.com');

        cy.get(selectors.buttons.save).should('be.disabled');

        cy.get(selectors.buttons.cancel).click();
    });

    it('should reject creating a second integration when one already exists (singleton enforcement)', () => {
        const firstIntegrationName = generateNameWithDate(`${aiIntegrationTestNamePrefix} First`);
        const secondIntegrationName = generateNameWithDate(`${aiIntegrationTestNamePrefix} Second`);

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        getInputByLabel('Integration name').clear().type(firstIntegrationName);
        getInputByLabel('Service URL').clear().type('https://lightspeed-1.example.com');

        saveCreatedIntegrationInForm(integrationSource, integrationType);

        cy.get(`${selectors.tableRowNameLink}:contains("${firstIntegrationName}")`).should('exist');

        clickCreateNewIntegrationInTable(integrationSource, integrationType);

        getInputByLabel('Integration name').clear().type(secondIntegrationName);
        getInputByLabel('Service URL').clear().type('https://lightspeed-2.example.com');

        cy.get(selectors.buttons.save).click();

        cy.get('.pf-v6-c-alert.pf-m-danger').should('be.visible');
        cy.get('.pf-v6-c-alert.pf-m-danger').contains('only one AI integration is allowed');

        cy.get(selectors.buttons.cancel).click();

        deleteIntegrationInTable(integrationSource, integrationType, firstIntegrationName);
        cy.get(`${selectors.tableRowNameLink}:contains("${firstIntegrationName}")`).should(
            'not.exist'
        );
    });
});
