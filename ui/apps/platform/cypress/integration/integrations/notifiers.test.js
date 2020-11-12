import { selectors } from '../../constants/IntegrationsPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';

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

    it('should show a hint about stored credentials for Google Cloud SCC', () => {
        cy.get(selectors.googleCloudSCCTile).click();
        cy.get(`${selectors.table.rows}:contains('Google Cloud SCC Test')`).click();
        cy.get('div:contains("Service Account Key"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Jira', () => {
        cy.get(selectors.jiraTile).click();
        cy.get(`${selectors.table.rows}:contains('Jira Test')`).click();
        cy.get('div:contains("Password or API Token"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Email', () => {
        cy.get(selectors.emailTile).click();
        cy.get(`${selectors.table.rows}:contains('Email Test')`).click();
        cy.get('div:contains("Password"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Splunk', () => {
        cy.get(selectors.splunkTile).click();
        cy.get(`${selectors.table.rows}:contains('Splunk Test')`).click();
        cy.get('div:contains("HTTP Event Collector Token"):last [alt="help"]').trigger(
            'mouseenter'
        );
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for PagerDuty', () => {
        cy.get(selectors.pagerDutyTile).click();
        cy.get(`${selectors.table.rows}:contains('PagerDuty Test')`).click();
        cy.get('div:contains("PagerDuty Integration Key"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    it('should show a hint about stored credentials for Generic Webhook', () => {
        cy.get(selectors.genericWebhookTile).click();
        cy.get(`${selectors.table.rows}:contains('Generic Webhook Test')`).click();
        cy.get('div:contains("Password"):last [alt="help"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });

    describe('AWS Security Hub notifier', () => {
        it('should show the AWS Security Hub notifier', () => {
            cy.get(selectors.awsSecurityHubTile).click();
            cy.get(`${selectors.modalHeader}:contains('Configure AWS Security Hub plugin')`);
            cy.get(`${selectors.resultsSection}:contains('No AWS Security Hub integrations')`);
        });

        it('should disable the save button if all the required fields are not filled out', () => {
            cy.get(selectors.awsSecurityHubTile).click();

            cy.get(`${selectors.resultsSection}:contains('No AWS Security Hub integrations')`);

            cy.get(selectors.buttons.new).click();
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

            cy.get(selectors.buttons.closePanel).click();
        });

        it('should allow you to configure a new AWS Security Hub integration when none exists', () => {
            cy.get(selectors.awsSecurityHubTile).click();

            cy.get(`${selectors.resultsSection}:contains('No AWS Security Hub integrations')`);

            cy.get(selectors.buttons.new).click();

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
            cy.get(selectors.awsSecurityHubForm.active).should('be.checked');

            cy.get(selectors.buttons.create).click();

            cy.get(`${selectors.toast.body}:contains("Successfully integrated AWS Security Hub")`);
        });
    });

    describe('Syslog notifier', () => {
        it('should show the Syslog notifier', () => {
            cy.get(selectors.syslogTile).click();
            cy.get(`${selectors.modalHeader}:contains('Configure Syslog plugin')`);
            cy.get(`${selectors.resultsSection}:contains('No Syslog integrations')`);
        });

        it('should disable the save button if all the required fields are not filled out', () => {
            cy.get(selectors.syslogTile).click();

            cy.get(selectors.buttons.new).click();
            cy.get(selectors.buttons.create).should('be.disabled'); // starts out disabled

            cy.get(selectors.syslogForm.nameInput).type('Test Syslog integration');
            cy.get(`${selectors.syslogForm.logFormat} button:first`).click();
            cy.get(selectors.syslogForm.localFacility).click();
            cy.get(`${selectors.syslogForm.localFacilityListItems}:contains('local7')`).click();
            cy.get(selectors.syslogForm.receiverHost).type('splunk.default');
            // not filling out the last required field, Receiver Port

            cy.get(selectors.buttons.create).should('be.disabled'); // still disabled

            cy.get(selectors.buttons.closePanel).click();
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
            cy.get(selectors.syslogForm.logFormat).click();
            cy.get(selectors.syslogForm.localFacility).click();
            cy.get(`${selectors.syslogForm.localFacilityListItems}:contains('local7')`).click();
            cy.get(selectors.syslogForm.receiverHost).type('splunk.default');
            cy.get(selectors.syslogForm.receiverPort).type('514');

            // test toggles, but then turn off again, to avoid actual TLS validation
            cy.get(selectors.syslogForm.useTls).click({ force: true });
            cy.get(selectors.syslogForm.disableTlsValidation).click({ force: true });
            cy.get(selectors.syslogForm.useTls).click({ force: true });
            cy.get(selectors.syslogForm.disableTlsValidation).click({ force: true });

            cy.get(selectors.syslogForm.active).should('be.checked');

            cy.get(selectors.buttons.create).click();

            cy.wait('@saveSyslogNotifier');

            cy.get(`${selectors.toast.body}:contains("Successfully integrated syslog")`, {
                timeout: 8000,
            });
        });
    });
});
