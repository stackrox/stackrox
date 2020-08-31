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
        it('should allow you to configure a new AWS Security Hub integration when none exists', () => {
            cy.get(selectors.awsSecurityHubTile).click();

            cy.get(`${selectors.resultsSection}:contains('No AWS Security Hub integrations')`);

            cy.get(selectors.buttons.new).click();

            cy.get(selectors.awsSecurityHubForm.nameInput);
            cy.get(selectors.awsSecurityHubForm.active);
            cy.get(selectors.awsSecurityHubForm.awsAccountNumber);
            cy.get(selectors.awsSecurityHubForm.awsRegionPlaceholder);
            cy.get(selectors.awsSecurityHubForm.awsAccessKeyId);
            cy.get(selectors.awsSecurityHubForm.awsSecretAccessKey);
        });
    });
});
