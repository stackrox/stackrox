import { selectors } from '../../constants/IntegrationsPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { editIntegration } from './integrationUtils';
import { getInputByLabel, getEscapedId } from '../../helpers/formHelpers';

describe('Notifiers Test', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route('GET', api.integrations.notifiers, 'fixture:integrations/notifiers.json').as(
            'getNotifiers'
        );

        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click({ force: true });
        cy.wait('@getNotifiers');
    });

    describe.skip('Notifier forms', () => {
        it('should create a new email integration', () => {
            cy.get(selectors.emailTile).click();

            // @TODO: only use the the click, and delete the direct URL visit after forms official launch
            cy.get(selectors.buttons.new).click();
            cy.visit('/main/integrations/notifiers/email/create');

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

            cy.get('#name-helper').contains('Required');
            cy.get(getEscapedId('email.server-helper')).contains('A server address is required');
            cy.get(getEscapedId('email.username-helper')).contains('A username is required');
            cy.get(getEscapedId('email.password-helper')).contains('A password is required');
            cy.get(getEscapedId('email.sender-helper')).contains(
                'A valid sender email address is required'
            );
            cy.get(getEscapedId('labelDefault-helper')).contains(
                'A default recipient email address is required'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            getInputByLabel('Email server').type('example.');
            getInputByLabel('Sender').type('scooby@doo', {
                parseSpecialCharSequences: false,
            });
            getInputByLabel('Default recipient')
                .type('shaggy@', {
                    parseSpecialCharSequences: false,
                })
                .blur();

            cy.get(getEscapedId('email.server-helper')).contains('Must be a valid server address');
            cy.get(getEscapedId('email.sender-helper')).contains(
                'A valid sender email address is required'
            );
            cy.get(getEscapedId('labelDefault-helper')).contains(
                'Must be a valid default recipient email address'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 3, check valid form form and save
            getInputByLabel('Integration name')
                .clear()
                .type(`Nova Email ${new Date().toISOString()}`);
            getInputByLabel('Email server').type('smtp.example.com:465');
            getInputByLabel('Username').clear().type('scooby');
            getInputByLabel('Password').clear().type('monkey');
            getInputByLabel('From').clear().type('ACS');
            getInputByLabel('Sender').clear().type('scooby@doo.com', {
                parseSpecialCharSequences: false,
            });
            getInputByLabel('Default recipient').clear().type('shaggy@example.com', {
                parseSpecialCharSequences: false,
            });
            getInputByLabel('Annotation key for recipient').clear().type('email');
            getInputByLabel('Disable TLS certificate validation (insecure)').click();

            cy.get(selectors.buttons.test).should('be.enabled');
            cy.get(selectors.buttons.save).should('be.enabled').click();
        });

        it('should create a new Google Cloud SCC integration', () => {
            cy.get(selectors.googleCloudSCCTile).click();

            // @TODO: only use the the click, and delete the direct URL visit after forms official launch
            cy.get(selectors.buttons.new).click();
            cy.visit('/main/integrations/notifiers/cscc/create');

            // Step 0, should start out with disabled Save and Test buttons
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 1, check empty fields
            getInputByLabel('Integration name').type(' ');
            getInputByLabel('Cloud SCC Source ID').type(' ');
            getInputByLabel('Service Account Key (JSON)').type(' ').blur();

            cy.get('#name-helper').contains('Required');
            cy.get(getEscapedId('cscc.sourceId-helper')).contains('A source ID is required');
            cy.get(getEscapedId('cscc.serviceAccount-helper')).contains(
                'A service account is required'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            getInputByLabel('Cloud SCC Source ID').type('organization-123');
            getInputByLabel('Service Account Key (JSON)')
                .type('{ "type": "service_account", "project_id": "123456"', {
                    parseSpecialCharSequences: false,
                })
                .blur();

            cy.get('#name-helper').contains('Required');
            cy.get(getEscapedId('cscc.sourceId-helper')).contains(
                'SCC source ID must match the format: organizations/[0-9]+/sources/[0-9]+'
            );
            cy.get(getEscapedId('cscc.serviceAccount-helper')).contains(
                'Service account must be valid JSON'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 3, check valid form form and save
            getInputByLabel('Integration name')
                .clear()
                .type(`Nova Google Cloud SCC ${new Date().toISOString()}`);
            getInputByLabel('Cloud SCC Source ID').clear().type('organizations/123/sources/456');
            getInputByLabel('Service Account Key (JSON)')
                .clear()
                .type('{ "type": "service_account", "project_id": "123456" }', {
                    parseSpecialCharSequences: false,
                })
                .blur();

            cy.get(selectors.buttons.test).should('be.enabled');
            cy.get(selectors.buttons.save).should('be.enabled').click();
        });

        it('should create a new splunk integration', () => {
            cy.get(selectors.splunkTile).click();

            // @TODO: only use the the click, and delete the direct URL visit after forms official launch
            cy.get(selectors.buttons.new).click();
            cy.visit('/main/integrations/notifiers/splunk/create');

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

            cy.get(getEscapedId('notifier.name-helper')).contains('Name is required');
            cy.get(getEscapedId('notifier.splunk.httpEndpoint-helper')).contains(
                'HTTP event collector URL is required'
            );
            cy.get(getEscapedId('notifier.splunk.httpToken-helper')).contains(
                'HTTP token is required'
            );
            cy.get(getEscapedId('notifier.splunk.truncate-helper')).contains(
                'HEC truncate limit is required'
            );
            cy.get(getEscapedId('notifier.splunk.sourceTypes.alert-helper')).contains(
                'Source type for alert is required'
            );
            cy.get(getEscapedId('notifier.splunk.sourceTypes.audit-helper')).contains(
                'Source type for audit is required'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 2, check fields for invalid formats
            getInputByLabel('HTTP event collector URL').clear().type('https://input').blur();

            cy.get(getEscapedId('notifier.splunk.httpEndpoint-helper')).contains(
                'Must be a valid server address'
            );
            cy.get(selectors.buttons.test).should('be.disabled');
            cy.get(selectors.buttons.save).should('be.disabled');

            // Step 3, check valid form form and save
            getInputByLabel('Integration name')
                .clear()
                .type(`Nova Email ${new Date().toISOString()}`);
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

            cy.get(selectors.buttons.test).should('be.enabled');
            cy.get(selectors.buttons.save).should('be.enabled').click();
        });
    });

    // @DEPRECATED: change this test after migrating forms to PatternFly
    it('should show a hint about stored credentials for Google Cloud SCC', () => {
        cy.get(selectors.googleCloudSCCTile).click();
        editIntegration('Google Cloud SCC Test');
        cy.get('div:contains("Service Account Key"):last [data-testid="help-icon"]').trigger(
            'mouseenter'
        );
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Jira', () => {
        cy.get(selectors.jiraTile).click();
        editIntegration('Jira Test');
        cy.get('div:contains("Password or API Token"):last [data-testid="help-icon"]').trigger(
            'mouseenter'
        );
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    // @DEPRECATED: change this test after migrating forms to PatternFly
    it('should show a hint about stored credentials for Email', () => {
        cy.get(selectors.emailTile).click();
        editIntegration('Email Test');
        cy.get('div:contains("Password"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    // @DEPRECATED: change this test after migrating forms to PatternFly
    it('should show a hint about stored credentials for Splunk', () => {
        cy.get(selectors.splunkTile).click();
        editIntegration('Splunk Test');
        cy.get('div:contains("HTTP Event Collector Token"):last [data-testid="help-icon"]').trigger(
            'mouseenter'
        );
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for PagerDuty', () => {
        cy.get(selectors.pagerDutyTile).click();
        editIntegration('PagerDuty Test');
        cy.get('div:contains("PagerDuty Integration Key"):last [data-testid="help-icon"]').trigger(
            'mouseenter'
        );
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Generic Webhook', () => {
        cy.get(selectors.genericWebhookTile).click();
        editIntegration('Generic Webhook Test');
        cy.get('div:contains("Password"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    describe('AWS Security Hub notifier', () => {
        it('should show the AWS Security Hub notifier', () => {
            cy.get(selectors.awsSecurityHubTile).click();
            cy.get('.pf-c-breadcrumb').contains('AWS Security Hub');
        });

        it('should disable the save button if all the required fields are not filled out', () => {
            cy.get(selectors.awsSecurityHubTile).click();

            cy.get(selectors.buttons.newIntegration).click();
            cy.get(selectors.buttons.create).should('be.disabled'); // starts out disabled

            cy.get(selectors.awsSecurityHubForm.nameInput).type('Test AWS Sec Hub integration');
            cy.get(selectors.awsSecurityHubForm.awsAccountNumber).type('939357552774');
            cy.get(selectors.awsSecurityHubForm.awsRegion).click();
            cy.get(
                `${selectors.awsSecurityHubForm.awsRegionListItems}:contains('us-east-2')`
            ).click();
            cy.get(selectors.awsSecurityHubForm.awsAccessKeyId).type('EXAMPLE7AKIAIOSFODNN');
            // not filling out the last field, Secret Acccess Key

            cy.get(selectors.buttons.create).should('be.disabled'); // still disabled
        });

        it('should allow you to configure a new AWS Security Hub integration', () => {
            cy.get(selectors.awsSecurityHubTile).click();

            cy.get(selectors.buttons.newIntegration).click();

            cy.get(selectors.awsSecurityHubForm.nameInput).type('Test AWS Sec Hub integration');
            cy.get(selectors.awsSecurityHubForm.awsAccountNumber).type('939357552774');
            cy.get(selectors.awsSecurityHubForm.awsRegion).click();
            cy.get(
                `${selectors.awsSecurityHubForm.awsRegionListItems}:contains('us-east-2')`
            ).click();
            cy.get(selectors.awsSecurityHubForm.awsAccessKeyId).type('EXAMPLE7AKIAIOSFODNN');
            cy.get(selectors.awsSecurityHubForm.awsSecretAccessKey).type(
                'EXAMPLEKEYwJalrXUtnFEMI/K7MDENG/bPxRfiCY'
            );

            cy.get(selectors.buttons.create).click();

            cy.get(`${selectors.toast.body}:contains("Successfully integrated AWS Security Hub")`);
        });
    });

    describe('Syslog notifier', () => {
        it('should show the Syslog notifier', () => {
            cy.get(selectors.syslogTile).click();
            cy.get('.pf-c-breadcrumb').contains('Syslog');
        });

        it('should disable the save button if all the required fields are not filled out', () => {
            cy.get(selectors.syslogTile).click();

            cy.get(selectors.buttons.new).click();
            cy.get(selectors.buttons.create).should('be.disabled'); // starts out disabled

            cy.get(selectors.syslogForm.nameInput).type('Test Syslog integration');
            cy.get(selectors.syslogForm.localFacility).click();
            cy.get(`${selectors.syslogForm.localFacilityListItems}:contains('local7')`).click();
            cy.get(selectors.syslogForm.receiverHost).type('splunk.default');
            // not filling out the last required field, Receiver Port

            cy.get(selectors.buttons.create).should('be.disabled'); // still disabled
        });

        it('should allow you to configure a new Syslog integration when none exists', () => {
            cy.route(
                'POST',
                api.integrations.notifiers,
                'fixture:integrations/syslogResponse.json'
            ).as('saveSyslogNotifier');
            cy.get(selectors.syslogTile).click();

            cy.get(selectors.buttons.new).click();

            cy.get(selectors.syslogForm.nameInput).type('Test Syslog integration');
            cy.get(selectors.syslogForm.localFacility).click();
            cy.get(`${selectors.syslogForm.localFacilityListItems}:contains('local7')`).click();
            cy.get(selectors.syslogForm.receiverHost).type('splunk.default');
            cy.get(selectors.syslogForm.receiverPort).type('514');

            // test toggles, but then turn off again, to avoid actual TLS validation
            cy.get(selectors.syslogForm.useTls).click({ force: true });
            cy.get(selectors.syslogForm.disableTlsValidation).click({ force: true });
            cy.get(selectors.syslogForm.useTls).click({ force: true });
            cy.get(selectors.syslogForm.disableTlsValidation).click({ force: true });

            cy.get(selectors.buttons.create).click();

            cy.wait('@saveSyslogNotifier');

            cy.get(`${selectors.toast.body}:contains("Successfully integrated syslog")`, {
                timeout: 8000,
            });
        });
    });
});
