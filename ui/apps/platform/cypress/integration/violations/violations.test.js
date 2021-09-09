import {
    url as violationsUrl,
    selectors as ViolationsPageSelectors,
} from '../../constants/ViolationsPage';
import { selectors as PoliciesPageSelectors } from '../../constants/PoliciesPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';

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
        const alertId = '8aaa344c-6266-4037-bc21-cd9323a54a4b';
        cy.intercept('GET', `/v1/alerts/${alertId}`, {
            fixture: 'alerts/alertById.json',
        }).as('alertById');
    };

    const mockGetAlertWithEmptyContainerConfig = () => {
        cy.fixture('alerts/alertWithEmptyContainerConfig.json').as('alertWithEmptyContainerConfig');
        cy.route('GET', api.alerts.alertById, '@alertWithEmptyContainerConfig').as(
            'alertWithEmptyContainerConfig'
        );
    };
    const mockExclusionDeployment = () => {
        cy.fixture('alerts/alertsWithExcludedDeployments.json').as('alertsWithExcludedDeployments');
        cy.route('GET', api.alerts.alerts, '@alertsWithExcludedDeployments').as(
            'alertsWithExcludedDeployments'
        );
    };

    const mockPatchAlerts = () => {
        cy.route({
            method: 'PATCH',
            url: '/v1/alerts/*',
            response: {},
        }).as('patchAlerts');
    };

    const mockGetPolicy = () => {
        cy.route({
            method: 'GET',
            url: '/v1/policies/*',
            response: {},
        }).as('getPolicy');
    };

    it('should select item in nav bar', () => {
        cy.get(ViolationsPageSelectors.navLink).should('have.class', 'pf-m-current');
    });

    it('should have violations in table', () => {
        cy.get(ViolationsPageSelectors.table.rows).should('have.length', 2);
    });

    it('should have Lifecycle column in table', () => {
        cy.get(ViolationsPageSelectors.table.column.lifecycle).should('be.visible');
        cy.get(ViolationsPageSelectors.firstTableRow).should('contain', 'Runtime');
    });

    it('should show the detail page on row click', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstTableRowLink).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.details.page).should('be.visible');
        cy.get(ViolationsPageSelectors.details.title).should('have.text', 'Misuse of iptables');
        cy.get(ViolationsPageSelectors.details.subtitle).should(
            'have.text',
            'in "ip-masq-agent" deployment'
        );
    });

    it('should have Entity column in table', () => {
        cy.get(ViolationsPageSelectors.table.column.entity).should('be.visible');
    });

    it('should have Type column in table', () => {
        cy.get(ViolationsPageSelectors.table.column.type).should('be.visible');
    });

    it('should have 4 tabs in the sidepanel', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstTableRowLink).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.details.tabs).should('have.length', 4);
        cy.get(ViolationsPageSelectors.details.violationTab).should('exist');
        cy.get(ViolationsPageSelectors.details.enforcementTab).should('exist');
        cy.get(ViolationsPageSelectors.details.deploymentTab).should('exist');
        cy.get(ViolationsPageSelectors.details.policyTab).should('exist');
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
        cy.get(
            `${ViolationsPageSelectors.firstTableRow} ${ViolationsPageSelectors.actions.btn}`
        ).click();
        cy.get(ViolationsPageSelectors.firstTableRow)
            .get(ViolationsPageSelectors.actions.excludeDeploymentBtn)
            .should('exist')
            .get(ViolationsPageSelectors.actions.resolveBtn)
            .should('exist')
            .get(ViolationsPageSelectors.actions.resolveAndAddToBaselineBtn)
            .should('exist');

        // to click out and reset the actions dropdown
        cy.get('body').type('{esc}');

        // Lifecycle: Deploy
        cy.get(
            `${ViolationsPageSelectors.lastTableRow} ${ViolationsPageSelectors.actions.btn}`
        ).click();
        cy.get(ViolationsPageSelectors.lastTableRow)
            .get(ViolationsPageSelectors.actions.resolveBtn)
            .should('not.exist')
            .get(ViolationsPageSelectors.actions.resolveAndAddToBaselineBtn)
            .should('not.exist')
            .get(ViolationsPageSelectors.actions.excludeDeploymentBtn)
            .should('exist');
    });

    // Excluding this test because it's causing issues. Will include it again once it's fixed in a different PR
    // also need to test bulk whitelisting (see ROX-2304)
    xit('should exclude the deployment', () => {
        mockExclusionDeployment();
        mockPatchAlerts();
        mockGetPolicy();
        cy.get(ViolationsPageSelectors.lastTableRow).find('[type="checkbox"]').check();
        cy.get('.panel-actions button').first().click();
        cy.get('.ReactModal__Content .btn.btn-success').click();
        cy.wait('@getPolicy');
        cy.visit('/main/violations');
        cy.wait('@alertsWithExcludedDeployments');
        cy.get(ViolationsPageSelectors.excludedDeploymentRow).should('not.exist');
    });

    it('should have enforcement information in the Enforcement tab', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstTableRowLink).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.details.enforcementTab).click();
        cy.get(ViolationsPageSelectors.enforcement.detailMessage).should((message) => {
            expect(message).to.contain('Kill Pod');
        });
        cy.get(ViolationsPageSelectors.enforcement.explanationMessage).should((message) => {
            expect(message).to.contain('Runtime data was evaluated against this StackRox policy');
        });
    });

    it('should have deployment information in the Deployment Details tab', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstTableRowLink).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.details.deploymentTab).click();
        cy.get(ViolationsPageSelectors.deployment.overview).should('have.exist');
        cy.get(ViolationsPageSelectors.deployment.containerConfiguration).should('exist');
        cy.get(ViolationsPageSelectors.deployment.securityContext).should('exist');
        cy.get(ViolationsPageSelectors.deployment.portConfiguration).should('exist');

        cy.get(ViolationsPageSelectors.deployment.snapshotWarning).should('exist');
    });

    it('should show deployment information in the Deployment Details tab with no container configuration values', () => {
        mockGetAlertWithEmptyContainerConfig();
        cy.get(ViolationsPageSelectors.lastTableRowLink).click();
        cy.wait('@alertWithEmptyContainerConfig');
        cy.get(ViolationsPageSelectors.details.deploymentTab).click();
        cy.get(
            `${ViolationsPageSelectors.deployment.containerConfiguration} [data-testid="commands"]`
        ).should('not.exist');
    });

    it('should have policy information in the Policy Details tab', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstTableRowLink).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.details.policyTab).click();
        cy.get(PoliciesPageSelectors.policyDetailsPanel.detailsSection).should('exist');
        cy.get(PoliciesPageSelectors.policyDetailsPanel.idValueDiv).should('exist');
    });

    it('should request the alerts in descending time order by default', () => {
        cy.get('@alerts')
            .its('url')
            .should('include', 'pagination.sortOption.field=Violation Time');
        cy.get('@alerts').its('url').should('include', 'pagination.sortOption.reversed=true');
    });

    it('should sort violations when clicking on a table header', () => {
        // first click will sort in direct order
        cy.get(ViolationsPageSelectors.table.column.policy).click();
        cy.wait('@alerts')
            .its('url')
            .should(
                'include',
                'pagination.sortOption.field=Policy&pagination.sortOption.reversed=false'
            );
        cy.get(ViolationsPageSelectors.firstTableRow).should('contain', 'ip-masq-agent');

        // second click will sort in reverse order
        cy.fixture('alerts/alerts.json').then((alertsData) => {
            const reverseSortedAlerts = {
                alerts: alertsData.alerts.sort(
                    (a, b) => -1 * a.policy.name.localeCompare(b.policy.name)
                ),
            };
            cy.route('GET', api.alerts.alerts, reverseSortedAlerts).as('alerts');
        });
        cy.get(ViolationsPageSelectors.table.column.policy).click();
        cy.wait('@alerts')
            .its('url')
            .should(
                'include',
                'pagination.sortOption.field=Policy&pagination.sortOption.reversed=true'
            );
        cy.get(ViolationsPageSelectors.firstTableRow).should('contain', 'metadata-proxy-v0.1');
    });
});
