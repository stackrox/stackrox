import {
    url as violationsUrl,
    selectors as ViolationsPageSelectors
} from '../constants/ViolationsPage';
import { selectors as PoliciesPageSelectors } from '../constants/PoliciesPage';
import * as api from '../constants/apiEndpoints';
import selectors from '../constants/SearchPage';
import withAuth from '../helpers/basicAuth';

describe('Violations page', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.fixture('alerts/alerts.json').as('alertsJson');
        cy.route('GET', api.alerts.alerts, '@alertsJson').as('alerts');

        cy.fixture('alerts/alertsCount.json').as('alertsCountJson');
        cy.route('GET', api.alerts.alertscount, '@alertsCountJson').as('alertsCount');

        cy.visit(violationsUrl);
        cy.wait('@alerts');
        cy.wait('@alertsCount');
    });

    const mockGetAlert = () => {
        cy.fixture('alerts/alertById.json').as('alertById');
        cy.route('GET', api.alerts.alertById, '@alertById').as('alertById');
    };

    const mockResolveAlert = () => {
        cy.route('PATCH', api.alerts.resolveAlert, {}).as('resolve');
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

    const mockPatchAlerts = () => {
        cy.route({
            method: 'PATCH',
            url: '/v1/alerts/*',
            response: {}
        }).as('patchAlerts');
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

    // TODO(ROX-3106)
    xit('should have 4 tabs in the sidepanel', () => {
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

    // @TODO: Figure out how to mock GraphQL, because this test depends on that working
    xit('should have a collapsible card for runtime violation', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .find(ViolationsPageSelectors.sidePanel.tabs)
            .get(ViolationsPageSelectors.sidePanel.getTabByIndex(0))
            .click();
        cy.get(ViolationsPageSelectors.runtimeProcessCards);
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

    // Excluding this test because it's causing issues. Will include it again once it's fixed in a different PR
    // also need to test bulk whitelisting (see ROX-2304)
    xit('should whitelist the deployment', () => {
        mockWhitelistDeployment();
        mockPatchAlerts();
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

    it('should have enforcement information in the Enforcement tab', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.panels, { timeout: 7000 })
            .eq(1)
            .get(ViolationsPageSelectors.sidePanel.getTabByIndex(1), { timeout: 7000 })
            .click();
        cy.get(ViolationsPageSelectors.sidePanel.enforcementDetailMessage).should(message => {
            expect(message).to.contain('Kill Pod');
        });
        cy.get(ViolationsPageSelectors.sidePanel.enforcementExplanationMessage).should(message => {
            expect(message).to.contain('Runtime data was evaluated against this StackRox policy');
        });
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
        cy.get(ViolationsPageSelectors.sidePanel.enforcementTab).click();
        cy.get(ViolationsPageSelectors.securityBestPractices).should('not.have.text', 'Commands');
    });

    it('should have policy information in the Policy Details tab', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.sidePanel.policyTab).click();
        cy.get(PoliciesPageSelectors.policyDetailsPanel.detailsSection).should('exist');
        cy.get(PoliciesPageSelectors.policyDetailsPanel.idValueDiv).should('exist');
    });

    it('should close side panel after resolving violation', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.panels).should('be.visible');

        mockResolveAlert();
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstPanelTableRow)
            .get(ViolationsPageSelectors.resolveButton)
            .eq(1)
            .click({ force: true });
        cy.wait('@resolve');

        cy.get(ViolationsPageSelectors.panels)
            .eq(1)
            .should('not.be.visible');
    });

    it('should request the alerts in descending time order by default', () => {
        cy.get('@alerts')
            .its('url')
            .should('include', 'pagination.sortOption.field=Violation Time');
        cy.get('@alerts')
            .its('url')
            .should('include', 'pagination.sortOption.reversed=true');
    });
});
