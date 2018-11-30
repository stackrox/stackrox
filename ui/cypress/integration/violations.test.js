import {
    url as violationsUrl,
    selectors as ViolationsPageSelectors
} from './constants/ViolationsPage';
import * as api from './constants/apiEndpoints';
import selectors from './constants/SearchPage';
import withAuth from './helpers/basicAuth';

describe('Violations page', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.fixture('alerts/alerts.json').as('alerts');
        cy.route('GET', api.alerts.alerts, '@alerts').as('alerts');
        cy.visit(violationsUrl);
        cy.wait('@alerts');
    });

    const mockGetAlert = () => {
        cy.fixture('alerts/alertById.json').as('alertById');
        cy.route('GET', api.alerts.alertById, '@alertById').as('alertById');
    };

    const mockGetAlertWithEmptyContainerConfig = () => {
        cy.fixture('alerts/alertWithEmptyContainerConfig.json').as('alertWithEmptyContainerConfig');
        cy.route('GET', api.alerts.alertById, '@alertWithEmptyContainerConfig').as(
            'alertWithEmptyContainerConfig'
        );
    };
    const mockWhitelistDeployment = () => {
        cy.fixture('alerts/alertsWithWhitelistedDeployments.json').as(
            'alertsWithWhitelistedDeployments'
        );
        cy.route('GET', api.alerts.alerts, '@alertsWithWhitelistedDeployments').as(
            'alertsWithWhitelistedDeployments'
        );
    };

    const mockGetPolicy = () => {
        cy.route({
            method: 'GET',
            url: '/v1/policies/*',
            response: {}
        }).as('getPolicy');
    };

    it('should select item in nav bar', () => {
        cy.get(ViolationsPageSelectors.navLink).should('have.class', 'bg-primary-700');
    });

    it('should have violations in table', () => {
        cy.get(ViolationsPageSelectors.rows).should('have.length', 2);
    });

    it('should have Lifecycle column in table', () => {
        cy.get(ViolationsPageSelectors.lifeCycleColumn).should('be.visible');
        cy.get(ViolationsPageSelectors.firstTableRow).should('contain', 'Runtime');
    });

    it('should show the side panel on row click', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .should('be.visible');
    });

    it('should show side panel with panel header', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstTableRow).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.header)
            .should('have.text', 'ip-masq-agent (70ee2b9a-c28c-11e8-b8c4-42010a8a0fe9)');
    });

    it('should have cluster column in table', () => {
        cy.get(ViolationsPageSelectors.clusterTableHeader).should('be.visible');
    });

    it('should close the side panel on search filter', () => {
        cy.visit(violationsUrl);
        cy.get(selectors.pageSearchInput).type('Cluster:{enter}', { force: true });
        cy.get(selectors.pageSearchInput).type('remote{enter}', { force: true });
        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .should('not.be.visible');
    });

    it('should have 4 tabs in the sidepanel', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.tabs)
            .should('have.length', 4);
        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.tabs)
            .eq(0)
            .should('have.text', 'Violation');
        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.tabs)
            .eq(1)
            .should('have.text', 'Enforcement');
        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.tabs)
            .eq(2)
            .should('have.text', 'Deployment');
        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.tabs)
            .eq(3)
            .should('have.text', 'Policy');
    });

    it('should have a collapsible card for runtime violation', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.tabs)
            .get(ViolationsPageSelectors.sidePanel.getTabByIndex(0))
            .click();
        cy.get(ViolationsPageSelectors.runtimeProcessCards).should('have.length', 10);
    });

    it('should contain correct action buttons for the lifecycle stage', () => {
        // Lifecycle: Runtime
        cy.get(ViolationsPageSelectors.firstTableRow)
            .get(ViolationsPageSelectors.whitelistDeploymentButton)
            .should('exist')
            .get(ViolationsPageSelectors.resolveButton)
            .should('exist');

        // Lifecycle: Deploy
        cy.get(ViolationsPageSelectors.lastTableRow)
            .get(ViolationsPageSelectors.resolveButton)
            .should('be.hidden')
            .get(ViolationsPageSelectors.whitelistDeploymentButton)
            .should('exist');
    });

    it('should whitelist the deployment', () => {
        mockWhitelistDeployment();
        mockGetPolicy();
        cy.get(ViolationsPageSelectors.lastTableRow)
            .find('[type="checkbox"]')
            .check();
        cy.get('.panel-actions button')
            .first()
            .click();
        cy.get('.ReactModal__Content .btn.btn-success').click();

        cy.wait('@getPolicy');
        cy.visit('/main/violations');
        cy.wait('@alertsWithWhitelistedDeployments');
        cy.get(ViolationsPageSelectors.whitelistDeploymentRow).should('not.exist');
    });

    it('should have deployment information in the Deployment Details tab', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .get(ViolationsPageSelectors.sidePanel.getTabByIndex(2))
            .click();
        cy.get(ViolationsPageSelectors.collapsible.header).should('have.length', 3);
        cy.get(ViolationsPageSelectors.collapsible.header)
            .eq(0)
            .should('have.text', 'Overview');
        cy.get(ViolationsPageSelectors.collapsible.header)
            .eq(1)
            .should('have.text', 'Container configuration');
        cy.get(ViolationsPageSelectors.collapsible.header)
            .eq(2)
            .should('have.text', 'Security Context');
    });

    it('should show deployment information in the Deployment Details tab with no container configuration values', () => {
        mockGetAlertWithEmptyContainerConfig();
        cy.get(ViolationsPageSelectors.lastTableRow).click();
        cy.wait('@alertWithEmptyContainerConfig');
        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .get(ViolationsPageSelectors.sidePanel.getTabByIndex(1))
            .click();
        cy.get(ViolationsPageSelectors.securityBestPractices).should('not.have.text', 'Commands');
    });
});
