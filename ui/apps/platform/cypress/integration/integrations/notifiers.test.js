import withAuth from '../../helpers/basicAuth';
import {
    generateNameWithDate,
    getHelperElementByLabel,
    getInputByLabel,
} from '../../helpers/formHelpers';
import sampleCert from '../../helpers/sampleCert';
import fakeGCPServiceAccount from '../../helpers/fakeGCPServiceAccount';

import {
    clickCreateNewIntegrationInTable,
    deleteIntegrationInTable,
    saveCreatedIntegrationInForm,
    testIntegrationInFormWithoutStoredCredentials,
    testIntegrationInFormWithStoredCredentials,
    visitIntegrationsTable,
} from './integrations.helpers';
import { selectors } from './integrations.selectors';

// Page address segments are the source of truth for integrationSource and integrationType.
const integrationSource = 'notifiers';

const staticResponseForTest = { body: {} };

const staticResponseForPOST = {
    body: { id: 'abcdefgh' },
};

describe('Notifier Integrations', () => {
    withAuth();

    describe('forms', () => {
        it('should create a new AWS Security Hub integration', () => {
            const integrationName = generateNameWithDate('Nova AWS Security Hub');
            const integrationType = 'awsSecurityHub';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').click().blur();
            getInputByLabel('AWS account number').click().blur();
            getInputByLabel('AWS region').focus().blur(); // focus, then blur, select in order to trigger validation
            getInputByLabel('Access key ID').click().blur();
            getInputByLabel('Secret access key').click().blur();

            getHelperElementByLabel('Integration name').contains('An integration name is required');
            getHelperElementByLabel('AWS account number').contains(
                'An AWS account number is required'
            );
            getHelperElementByLabel('AWS region').contains('An AWS region is required');
            getHelperElementByLabel('Access key ID').contains('An access key ID is required');
            getHelperElementByLabel('Secret access key').contains(
                'A secret access key is required'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check conditional fields

            // Step 2.1, enable container IAM role, this should remove the AWS credentials fields
            getInputByLabel('Use container IAM role').click();
            cy.get(
                `.pf-c-form__group:has('.pf-c-form__control:contains("Access key ID")') input`
            ).should('not.exist');
            cy.get(
                `.pf-c-form__group:has('.pf-c-form__control:contains("Secret access key")') input`
            ).should('not.exist');
            // Step 2.2, disable container IAM role, this should render the AWS credentials fields again
            getInputByLabel('Use container IAM role').click();
            getInputByLabel('Access key ID').should('be.visible');
            getInputByLabel('Secret access key').should('be.visible');

            // Step 3, check fields for invalid formats
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('AWS region').select('US East (N. Virginia) us-east-1');
            getInputByLabel('Access key ID').click().type('AKIA5VNQSYCDODH7VKMK');
            getInputByLabel('Secret access key')
                .click()
                .type('3JBA+EtbcGwONcx+1CKvbCn4FxFLRGiDANfzD+Vr');
            getInputByLabel('AWS account number').clear().type('93935755277').blur();

            getHelperElementByLabel('AWS account number').contains(
                'AWS account numbers must be 12 characters long'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 4, check valid form and save
            getInputByLabel('AWS account number').clear().type('939357552771').blur();

            testIntegrationInFormWithStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            saveCreatedIntegrationInForm(integrationSource, integrationType);

            deleteIntegrationInTable(integrationSource, integrationType, integrationName);
        });

        it('should create a new Email integration', () => {
            const integrationName = generateNameWithDate('Nova Email');
            const integrationType = 'email';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').type(' ');
            getInputByLabel('Email server').type(' ');
            getInputByLabel('Username').type(' ');
            getInputByLabel('Password').type(' ');
            getInputByLabel('Sender').type(' ');
            getInputByLabel('Default recipient').type(' ').blur();

            getHelperElementByLabel('Integration name').contains(
                'Email integration name is required'
            );
            getHelperElementByLabel('Email server').contains('A server address is required');
            getHelperElementByLabel('Username').contains('A username is required');
            getHelperElementByLabel('Password').contains('A password is required');
            getHelperElementByLabel('Sender').contains('A sender email address is required');
            getHelperElementByLabel('Default recipient').contains(
                'A default recipient email address is required'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            getInputByLabel('Email server').type('example.');
            getInputByLabel('Sender').type('scooby@doo.', {
                parseSpecialCharSequences: false,
            });
            getInputByLabel('Default recipient')
                .type('shaggy@', {
                    parseSpecialCharSequences: false,
                })
                .blur();

            getHelperElementByLabel('Email server').contains('Must be a valid server address');
            getHelperElementByLabel('Sender').contains('Must be a valid sender email address');
            getHelperElementByLabel('Default recipient').contains(
                'Must be a valid default recipient email address'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 3, check valid from and save
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('Email server').type('smtp.example.com:465');
            getInputByLabel('Username').clear().type('scooby');
            getInputByLabel('Password').clear().type('monkey');
            getInputByLabel('From').clear().type('Scooby');
            getInputByLabel('Sender').clear().type('scooby@doo.com', {
                parseSpecialCharSequences: false,
            });
            getInputByLabel('Default recipient').clear().type('shaggy@example.com', {
                parseSpecialCharSequences: false,
            });
            getInputByLabel('Annotation key for recipient').clear().type('email');
            getInputByLabel('Disable TLS certificate validation (insecure)').click();

            testIntegrationInFormWithStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            saveCreatedIntegrationInForm(integrationSource, integrationType);

            deleteIntegrationInTable(integrationSource, integrationType, integrationName);
        });

        it('should create a new Generic Webhook integration', () => {
            const integrationName = generateNameWithDate('Nova Generic Webhook');
            const integrationType = 'generic';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').type(' ');
            getInputByLabel('Endpoint').type(' ').blur();
            getHelperElementByLabel('Integration name').contains('Name is required');
            getHelperElementByLabel('Endpoint').contains('Endpoint is required');
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats, or conditional validation
            getInputByLabel('Endpoint').type('example.com:3000/hooks/123').blur();

            getInputByLabel('Username').type('neo').blur();
            getInputByLabel('Password').type(' ').blur();
            getHelperElementByLabel('Password').contains(
                'A password is required if the integration has a username'
            );

            getInputByLabel('Password').clear().type('monkey').blur();
            getInputByLabel('Username').clear().type(' ').blur();
            getHelperElementByLabel('Username').contains(
                'A username is required if the integration has a password'
            );

            // Step 3, check valid from and save
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('Endpoint').clear().type('service-name');
            getInputByLabel('CA certificate (optional)').type(sampleCert, { delay: 1 });
            getInputByLabel('Skip TLS verification').click();
            getInputByLabel('Enable audit logging').click();
            getInputByLabel('Username').clear().type('neo');
            getInputByLabel('Password').clear().type('spoon').blur();
            cy.get('button:contains("Add new header")').click();
            getInputByLabel('Key').type('x-org');
            getInputByLabel('Value').type('mysteryinc').blur();

            testIntegrationInFormWithStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            // Mock response to prevent error in central log file: Unable to send audit msg.
            saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

            // Test does not delete, because it did not create.
        });

        it('should create a new Google Cloud SCC integration', () => {
            const integrationName = generateNameWithDate('Nova Google Cloud SCC');
            const integrationType = 'cscc';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').type(' ');
            getInputByLabel('Cloud SCC Source ID').type(' ');
            getInputByLabel('Service account key (JSON)').type(' ').blur();

            getHelperElementByLabel('Integration name').contains('An integration name is required');
            getHelperElementByLabel('Cloud SCC Source ID').contains('A source ID is required');
            getHelperElementByLabel('Service account key (JSON)').contains(
                'Valid JSON is required for service account key'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check conditional fields

            // Step 2.1, enable workload identity, this should remove the service account field
            getInputByLabel('Use workload identity').click();
            getInputByLabel('Service account key (JSON)').should('be.disabled');
            // Step 2.2, disable workload identity, this should render the service account field again
            getInputByLabel('Use workload identity').click();
            getInputByLabel('Service account key (JSON)').should('be.enabled');

            // Step 3, check fields for invalid formats
            getInputByLabel('Cloud SCC Source ID').type('organization-123');
            getInputByLabel('Service account key (JSON)')
                .type('{ "type": "service_account", "project_id": "123456"', {
                    parseSpecialCharSequences: false,
                })
                .blur();

            getHelperElementByLabel('Cloud SCC Source ID').contains(
                'SCC source ID must match the format: organizations/[0-9]+/sources/[0-9]+'
            );
            getHelperElementByLabel('Service account key (JSON)').contains(
                'Valid JSON is required for service account key'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 4, check valid from and save
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('Cloud SCC Source ID').clear().type('organizations/123/sources/456');
            getInputByLabel('Service account key (JSON)')
                .clear()
                .type(JSON.stringify(fakeGCPServiceAccount), { parseSpecialCharSequences: false })
                .blur();

            testIntegrationInFormWithStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            saveCreatedIntegrationInForm(integrationSource, integrationType);

            deleteIntegrationInTable(integrationSource, integrationType, integrationName);
        });

        it('should create a new Jira integration', () => {
            const integrationName = generateNameWithDate('Nova Jira');
            const integrationType = 'jira';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').click();
            getInputByLabel('Username').click();
            getInputByLabel('Password or API token').click();
            getInputByLabel('Issue type').click();
            getInputByLabel('Jira URL').click();
            getInputByLabel('Default project').click().blur();

            getHelperElementByLabel('Integration name').contains('Name is required');
            getHelperElementByLabel('Username').contains('Username is required');
            getHelperElementByLabel('Password or API token').contains(
                'Password or API token is required'
            );
            getHelperElementByLabel('Issue type').contains('Issue type is required');
            getHelperElementByLabel('Jira URL').contains('Jira URL is required');
            getHelperElementByLabel('Default project').contains('A default project is required');
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            // not certain if any fields have invalid formats at this time

            // Step 3, check valid form and save
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('Username').clear().type('socrates');
            getInputByLabel('Password or API token').clear().type('monkey');
            getInputByLabel('Issue type').clear().type('Bug');
            getInputByLabel('Jira URL').clear().type('https://example.atlassian.net');
            getInputByLabel('Default project').clear().type('Unicorn').blur();

            testIntegrationInFormWithStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            // Mock request because backend pings your Jira on Save, not just on Test.
            saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

            // Test does not delete, because it did not create.
        });

        it('should create a new PagerDuty integration', () => {
            const integrationName = generateNameWithDate('Nova PagerDuty');
            const integrationType = 'pagerduty';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').type(' ').blur();
            getInputByLabel('PagerDuty integration key').type(' ').clear().blur();

            getHelperElementByLabel('Integration name').contains('Integration name is required');
            getHelperElementByLabel('PagerDuty integration key').contains(
                'PagerDuty integration key is required'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            /*
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');
            */

            // Step 3, check valid form and save
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('PagerDuty integration key').type('key');

            testIntegrationInFormWithoutStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            saveCreatedIntegrationInForm(integrationSource, integrationType);

            deleteIntegrationInTable(integrationSource, integrationType, integrationName);
        });

        it('should create a new Sumo Logic integration', () => {
            const integrationName = generateNameWithDate('Nova Sumo Logic');
            const integrationType = 'sumologic';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').type(' ');
            getInputByLabel('HTTP Collector Source Address').type(' ').blur();

            getHelperElementByLabel('Integration name').contains('Integration name is required');
            getHelperElementByLabel('HTTP Collector Source Address').contains(
                'HTTP Collector Source Address is required'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            /*
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');
            */

            // Step 3, check valid from and save
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('HTTP Collector Source Address')
                .clear()
                .type('https://endpoint.sumologic.com/receiver/v1/http/');

            testIntegrationInFormWithoutStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            saveCreatedIntegrationInForm(integrationSource, integrationType);

            deleteIntegrationInTable(integrationSource, integrationType, integrationName);
        });

        it('should create a new Splunk integration', () => {
            const integrationName = generateNameWithDate('Nova Splunk');
            const integrationType = 'splunk';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').type(' ');
            getInputByLabel('HTTP event collector URL').type(' ');
            getInputByLabel('HTTP event collector token').type(' ');
            getInputByLabel('HEC truncate limit').clear().type(' ');
            getInputByLabel('Source type for alert').clear().type(' ');
            getInputByLabel('Source type for audit').clear().type(' ').blur();

            getHelperElementByLabel('Integration name').contains('Name is required');
            getHelperElementByLabel('HTTP event collector URL').contains(
                'HTTP event collector URL is required'
            );
            getHelperElementByLabel('HTTP event collector token').contains(
                'HTTP token is required'
            );
            getHelperElementByLabel('HEC truncate limit').contains(
                'HEC truncate limit is required'
            );
            getHelperElementByLabel('Source type for alert').contains(
                'Source type for alert is required'
            );
            getHelperElementByLabel('Source type for audit').contains(
                'Source type for audit is required'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            getInputByLabel('HTTP event collector URL').clear().type('https://input').blur();

            getHelperElementByLabel('HTTP event collector URL').contains(
                'Must be a valid server address'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 3, check valid from and save
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('HTTP event collector URL')
                .clear()
                .type(
                    'https://input-prd-p-76sv8wzbfpdv.cloud.splunk.com:8088/services/collector/event'
                );
            getInputByLabel('HTTP event collector token').clear().type('asecrettoken');
            getInputByLabel('HEC truncate limit').type('5000');
            getInputByLabel('Disable TLS certificate validation (insecure)').click();
            getInputByLabel('Enable audit logging').click();
            getInputByLabel('Source type for alert').clear().type('stackrox-alert');
            getInputByLabel('Source type for audit').clear().type('stackrox-audit-message');

            testIntegrationInFormWithStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            // Mock response to prevent error in central log file: Unable to send audit msg.
            saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

            // Test does not delete, because it did not create.
        });

        it('should create a new Slack integration', () => {
            const integrationName = generateNameWithDate('Nova Slack');
            const integrationType = 'slack';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').click().blur();
            getInputByLabel('Default Slack webhook').click().blur();
            getInputByLabel('Annotation key for Slack webhook').click().blur();

            getHelperElementByLabel('Integration name').contains('Name is required');
            getHelperElementByLabel('Default Slack webhook').contains('Slack webhook is required');
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('Default Slack webhook')
                .clear()
                .type('https://hooks.slack.com/services/')
                .blur();

            getHelperElementByLabel('Default Slack webhook').contains(
                'Must be a valid Slack webhook URL, like https://hooks.slack.com/services/EXAMPLE'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 3, check valid form and save
            getInputByLabel('Annotation key for Slack webhook').clear().type('slack');
            getInputByLabel('Default Slack webhook')
                .clear()
                .type('https://hooks.slack.com/services/scooby/doo')
                .blur();

            testIntegrationInFormWithoutStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            saveCreatedIntegrationInForm(integrationSource, integrationType);

            deleteIntegrationInTable(integrationSource, integrationType, integrationName);
        });

        it('should create a new Syslog integration', () => {
            const integrationName = generateNameWithDate('Nova Syslog');
            const integrationType = 'syslog';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').click().blur();
            getInputByLabel('Logging facility').focus().blur(); // focus, then blur, select in order to trigger validation
            getInputByLabel('Receiver host').click().blur();
            getInputByLabel('Receiver port').click().clear().blur();

            // check format toggle
            cy.get('.pf-c-form__group-label .pf-c-form__label-text:contains("Message Format")');
            cy.get(
                '#messageFormat .pf-c-toggle-group__item .pf-c-toggle-group__button.pf-m-selected:contains("CEF")'
            );

            getHelperElementByLabel('Integration name').contains('Integration name is required');
            getHelperElementByLabel('Logging facility').contains('Logging facility is required');
            getHelperElementByLabel('Receiver host').contains('Receiver host is required');
            getHelperElementByLabel('Receiver port').contains('Receiver port is required');
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('Logging facility').select('local0').blur();
            getInputByLabel('Receiver host').clear().type('host.example.com').blur();
            getInputByLabel('Receiver port').clear().type('65536').blur();

            getHelperElementByLabel('Receiver port').contains(
                'Receiver port must be between 1 and 65535'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 3, check valid form and save
            getInputByLabel('Receiver port').clear().type('1').blur();
            cy.get('button:contains("Add new extra field")').click();
            getInputByLabel('Key').type('vehicle');
            getInputByLabel('Value').type('vanagon').blur();
            cy.get(
                '#messageFormat .pf-c-toggle-group__item .pf-c-toggle-group__button:contains("CEF (legacy field order)")'
            ).click();
            cy.get(
                '#messageFormat .pf-c-toggle-group__item .pf-c-toggle-group__button:contains("CEF (legacy field order)")'
            ).should('have.class', 'pf-m-selected');

            testIntegrationInFormWithoutStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            // Mock response to prevent error in central log file: Unable to send audit msg.
            saveCreatedIntegrationInForm(integrationSource, integrationType, staticResponseForPOST);

            // Test does not delete, because it did not create.
        });

        it('should create a new Teams integration', () => {
            const integrationName = generateNameWithDate('Nova Teams');
            const integrationType = 'teams';

            visitIntegrationsTable(integrationSource, integrationType);
            clickCreateNewIntegrationInTable(integrationSource, integrationType);

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').click().blur();
            getInputByLabel('Default Teams webhook').click().blur();

            getHelperElementByLabel('Integration name').contains('Integration name is required');
            getHelperElementByLabel('Default Teams webhook').contains('Webhook is required');
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            // none

            // Step 3, check valid form and save
            getInputByLabel('Integration name').clear().type(integrationName);
            getInputByLabel('Default Teams webhook')
                .clear()
                .type('https://outlook.office365.com/webhook/scooby/doo')
                .blur();
            getInputByLabel('Annotation key for Teams webhook').clear().type('teams');

            testIntegrationInFormWithoutStoredCredentials(
                integrationSource,
                integrationType,
                staticResponseForTest
            );

            saveCreatedIntegrationInForm(integrationSource, integrationType);

            deleteIntegrationInTable(integrationSource, integrationType, integrationName);
        });
    });
});
