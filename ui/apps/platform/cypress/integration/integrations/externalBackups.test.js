import withAuth from '../../helpers/basicAuth';
import {
    generateNameWithDate,
    getHelperElementByLabel,
    getInputByLabel,
} from '../../helpers/formHelpers';

import {
    clickCreateNewIntegrationInTable,
    deleteIntegrationInTable,
    saveCreatedIntegrationInForm,
    testIntegrationInFormWithStoredCredentials,
    visitIntegrationsTable,
    visitIntegrationsWithStaticResponseForCapabilities,
    visitIntegrationsAndVerifyRedirectWithStaticResponseForCapabilities,
} from './integrations.helpers';
import { selectors } from './integrations.selectors';

// Page address segments are the source of truth for integrationSource and integrationType.
const integrationSource = 'backups';

describe('Backup Integrations', () => {
    withAuth();

    describe('forms', () => {
        it('should create a new S3 integration', () => {
            const integrationName = generateNameWithDate('Nova S3 Backup');
            const integrationType = 's3';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

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

            const staticResponseForTest = { body: {} };
            testIntegrationInFormWithStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            saveCreatedIntegrationInForm(integrationSource, integrationType);

            deleteIntegrationInTable(integrationSource, integrationType, integrationName);
        });

        it('should create a new Google Cloud Storage integration', () => {
            const integrationName = generateNameWithDate('Nova Google Cloud Backup');
            const integrationType = 'gcs';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').type(' ');
            getInputByLabel('Backups to retain').clear(); // clear the default value of 1
            getInputByLabel('Bucket').type(' ');
            getInputByLabel('Service account key (JSON)').type(' ').blur();

            getHelperElementByLabel('Integration name').contains('Integration name is required');
            getHelperElementByLabel('Backups to retain').contains(
                'Number of backups to keep is required'
            );
            getHelperElementByLabel('Bucket').contains('Bucket is required');
            getHelperElementByLabel('Service account key (JSON)').contains(
                'Valid JSON is required for service account key'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('Bucket').type('stackrox');
            getInputByLabel('Backups to retain').type('0').blur(); // enter too low a value
            getInputByLabel('Service account key (JSON)').type('{').blur(); // enter invalid JSON

            getHelperElementByLabel('Backups to retain').contains(
                'Number of backups to keep must be 1 or greater'
            );
            getHelperElementByLabel('Service account key (JSON)').contains(
                'Valid JSON is required for service account key'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 3, check conditional fields

            // Step 3.1, enable workload identity, this should remove the service account field
            getInputByLabel('Use workload identity').click();
            getInputByLabel('Service account key (JSON)').should('be.disabled');
            // Step 3.2, disable workload identity, this should render the service account field again
            getInputByLabel('Use workload identity').click();
            getInputByLabel('Service account key (JSON)').should('be.enabled');

            // Step 4, check valid from and save
            getInputByLabel('Use workload identity').click().click(); // clear service account, then re-enable it
            getInputByLabel('Object prefix').clear().type('acs-');
            getInputByLabel('Backups to retain').clear().type(1).blur();
            getInputByLabel('Service account key (JSON)')
                .type('{ "type": "service_account" }', {
                    parseSpecialCharSequences: false,
                })
                .blur(); // enter invalid JSON

            const staticResponseForTest = { body: {} };
            testIntegrationInFormWithStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            saveCreatedIntegrationInForm(integrationSource, integrationType);

            deleteIntegrationInTable(integrationSource, integrationType, integrationName);
        });
    });

    describe('when cloud backup capability is disabled', () => {
        it('should not render Image back integrations', () => {
            visitIntegrationsWithStaticResponseForCapabilities({
                body: { centralCanUseCloudBackupIntegrations: 'CapabilityDisabled' },
            });
            cy.get('h2:contains("Backup Integrations")').should('not.exist');
            cy.get('a .pf-c-card__title:contains("Amazon S3")').should('not.exist');
            cy.get('a .pf-c-card__title:contains("Google Cloud Storage")').should('not.exist');

            visitIntegrationsAndVerifyRedirectWithStaticResponseForCapabilities(
                {
                    body: { centralCanUseCloudBackupIntegrations: 'CapabilityDisabled' },
                },
                'backups',
                's3'
            );

            visitIntegrationsAndVerifyRedirectWithStaticResponseForCapabilities(
                {
                    body: { centralCanUseCloudBackupIntegrations: 'CapabilityDisabled' },
                },
                'backups',
                'gcs'
            );
        });
    });
});
