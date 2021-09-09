import { selectors } from '../../constants/IntegrationsPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { editIntegration } from './integrationUtils';
import { getHelperElementByLabel, getInputByLabel } from '../../helpers/formHelpers';

describe('External Backups Test', () => {
    withAuth();

    beforeEach(() => {
        cy.intercept('GET', api.integrations.externalBackups, {
            fixture: 'integrations/externalBackups.json',
        }).as('getExternalBackups');

        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click({ force: true });
        cy.wait('@getExternalBackups');
    });
    describe('External Backup forms', () => {
        it('should create a new S3 integration', () => {
            cy.get(selectors.amazonS3Tile).click();

            // @TODO: only use the the click, and delete the direct URL visit after forms official launch
            cy.get(selectors.buttons.new).click();
            cy.visit('/main/integrations/backups/s3/create');

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
            getInputByLabel('Integration name')
                .clear()
                .type(`Nova S3 Backup ${new Date().toISOString()}`);
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
            cy.get(selectors.buttons.save).should('be.enabled').click();

            cy.wait('@getExternalBackups');

            cy.location().should((loc) => {
                expect(loc.pathname).to.eq('/main/integrations/backups/s3');
            });
        });

        it('should create a new Google Cloud Storage integration', () => {
            cy.get(selectors.googleCloudStorageTile).click();

            // @TODO: only use the the click, and delete the direct URL visit after forms official launch
            cy.get(selectors.buttons.new).click();
            cy.visit('/main/integrations/backups/gcs/create');

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
            getInputByLabel('Integration name')
                .clear()
                .type(`Nova Google Cloud Backup ${new Date().toISOString()}`);
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
            cy.get(selectors.buttons.save).should('be.enabled').click();

            cy.wait('@getExternalBackups');

            cy.location().should((loc) => {
                expect(loc.pathname).to.eq('/main/integrations/backups/gcs');
            });
        });
    });

    it('should show a hint about stored credentials for Amazon S3', () => {
        cy.get(selectors.amazonS3Tile).click();
        editIntegration('Amazon S3 Test');
        cy.get('div:contains("Access Key ID"):last [data-testid="help-icon"]').trigger(
            'mouseenter'
        );
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
        cy.get('div:contains("Secret Access Key"):last [data-testid="help-icon"]').trigger(
            'mouseenter'
        );
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Google Cloud Storage', () => {
        cy.get(selectors.googleCloudStorageTile).click();
        editIntegration('Google Cloud Storage Test');
        cy.get('div:contains("Service Account JSON"):last [data-testid="help-icon"]').trigger(
            'mouseenter'
        );
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });
});
