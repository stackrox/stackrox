import * as api from '../../constants/apiEndpoints';
import { labels, selectors, url } from '../../constants/IntegrationsPage';
import withAuth from '../../helpers/basicAuth';
import {
    generateNameWithDate,
    getHelperElementByLabel,
    getInputByLabel,
} from '../../helpers/formHelpers';
import { visitIntegrationsUrl } from '../../helpers/integrations';

function assertBackupIntegrationTable(integrationType) {
    const label = labels.backups[integrationType];
    cy.get(`${selectors.breadcrumbItem}:contains("${label}")`);
    cy.get(`${selectors.title2}:contains("${label}")`);
}

function getBackupIntegrationTypeUrl(integrationType) {
    return `${url}/backups/${integrationType}`;
}

function visitBackupIntegrationType(integrationType) {
    visitIntegrationsUrl(getBackupIntegrationTypeUrl(integrationType));
    cy.intercept('GET', api.integrations.apiTokens).as('getAPITokens');
    cy.intercept('GET', api.integrations.clusterInitBundles).as('getClusterInitBundles');
    cy.intercept('GET', api.integrations.externalBackups).as('getBackupIntegrations');
    cy.visit(getBackupIntegrationTypeUrl(integrationType));
    cy.wait('@getBackupIntegrations');
    assertBackupIntegrationTable(integrationType);
}

function saveBackupIntegrationType(integrationType) {
    cy.intercept('GET', api.integrations.externalBackups).as('getBackupIntegrations');
    cy.intercept('POST', api.integrations.externalBackups).as('postBackupIntegration');
    cy.get(selectors.buttons.save).should('be.enabled').click();
    cy.wait(['@postBackupIntegration', '@getBackupIntegrations']);
    assertBackupIntegrationTable(integrationType);
    cy.location('pathname').should('eq', getBackupIntegrationTypeUrl(integrationType));
}

describe('External Backups Test', () => {
    withAuth();

    describe('External Backup forms', () => {
        it('should create a new S3 integration', () => {
            const integrationName = generateNameWithDate('Nova S3 Backup');
            const integrationType = 's3';
            visitBackupIntegrationType(integrationType);
            cy.get(selectors.buttons.newIntegration).click();

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').type(' ');
            getInputByLabel('Backups to retain').clear(); // clear the default value of 1
            getInputByLabel('Bucket').type(' ');
            getInputByLabel('Region').type(' ');
            getInputByLabel('Access key ID').type(' ');
            getInputByLabel('Secret access key').type(' ').blur();

            getHelperElementByLabel('Integration name').contains('Integration name is required');
            getHelperElementByLabel('Backups to retain').contains(
                'Number of backups to keep is required'
            );
            getHelperElementByLabel('Bucket').contains('Bucket is required');
            getHelperElementByLabel('Region').contains('Region is required');
            getHelperElementByLabel('Access key ID').contains('An access key ID is required');
            getHelperElementByLabel('Secret access key').contains(
                'A secret access key is required'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('Bucket').type('stackrox');
            getInputByLabel('Region').type('us-west-2');
            getInputByLabel('Use container IAM role').click();
            getInputByLabel('Backups to retain').type('0').blur(); // enter too low a value

            getHelperElementByLabel('Backups to retain').contains(
                'Number of backups to keep must be 1 or greater'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 3, check valid from and save
            getInputByLabel('Object prefix').clear().type('acs-');
            getInputByLabel('Endpoint').clear().type('s3.us-west-2.amazonaws.com');
            getInputByLabel('Backups to retain').clear().type(1).blur();

            cy.get(selectors.buttons.test).should('be.enabled');
            saveBackupIntegrationType(integrationType);
        });

        it('should create a new Google Cloud Storage integration', () => {
            const integrationName = generateNameWithDate('Nova Google Cloud Backup');
            const integrationType = 'gcs';
            visitBackupIntegrationType(integrationType);
            cy.get(selectors.buttons.newIntegration).click();

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').type(' ');
            getInputByLabel('Backups to retain').clear(); // clear the default value of 1
            getInputByLabel('Bucket').type(' ');
            getInputByLabel('Service account (JSON)').type(' ').blur();

            getHelperElementByLabel('Integration name').contains('Integration name is required');
            getHelperElementByLabel('Backups to retain').contains(
                'Number of backups to keep is required'
            );
            getHelperElementByLabel('Bucket').contains('Bucket is required');
            getHelperElementByLabel('Service account (JSON)').contains(
                'Valid JSON is required for service account'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('Bucket').type('stackrox');
            getInputByLabel('Backups to retain').type('0').blur(); // enter too low a value
            getInputByLabel('Service account (JSON)').type('{').blur(); // enter invalid JSON

            getHelperElementByLabel('Backups to retain').contains(
                'Number of backups to keep must be 1 or greater'
            );
            getHelperElementByLabel('Service account (JSON)').contains(
                'Valid JSON is required for service account'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 3, check valid from and save
            getInputByLabel('Use workload identity').click().click(); // clear service account, then re-enable it
            getInputByLabel('Object prefix').clear().type('acs-');
            getInputByLabel('Backups to retain').clear().type(1).blur();
            getInputByLabel('Service account (JSON)')
                .type('{ "type": "service_account" }', {
                    parseSpecialCharSequences: false,
                })
                .blur(); // enter invalid JSON

            cy.get(selectors.buttons.test).should('be.enabled');
            saveBackupIntegrationType(integrationType);
        });
    });
});
