import {
    url as violationsUrl,
    selectors as ViolationsPageSelectors,
} from '../../constants/ViolationsPage';
import { selectors as PoliciesPageSelectors } from '../../constants/PoliciesPage';
// import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';

// TODO delete search wildcards in apiEndpoints.js after all occurrences are in intercept calls.
const api = {
    alerts: {
        alerts: '/v1/alerts', // edit in apiEndpoints
        alertscount: '/v1/alertscount', // edit in apiEndpoints
        resolveAlert: '/v1/alerts/*/resolve', // already correct in apiEndpoints
    },
    risks: {
        getDeployment: '/v1/deployments/*', // already correct in apiEndpoints
    },
    policies: {
        policy: '/v1/policies/*', // already correct in apiEndpoints
    },
};

describe('Violations page', () => {
    withAuth();

    beforeEach(() => {
        cy.intercept('GET', `${api.alerts.alerts}?query=*`, {
            fixture: 'alerts/alerts.json',
        }).as('alerts');

        cy.intercept('GET', `${api.alerts.alertscount}?query=`, {
            fixture: 'alerts/alertsCount.json',
        }).as('alertsCount');

        cy.visit(violationsUrl);
        cy.wait('@alerts');
        cy.wait('@alertsCount');
    });

    const mockGetAlert = () => {
        const alertId = '8aaa344c-6266-4037-bc21-cd9323a54a4b';
        cy.intercept('GET', `${api.alerts.alerts}/${alertId}`, {
            fixture: 'alerts/alertById.json',
        }).as('alertById');
    };

    const mockGetAlertWithEmptyContainerConfig = () => {
        const alertId = '83f1d8d0-1e2b-410a-b1c3-c77ae2bb5ad9';
        cy.intercept('GET', `${api.alerts.alerts}/${alertId}`, {
            fixture: 'alerts/alertWithEmptyContainerConfig.json',
        }).as('alertWithEmptyContainerConfig');
    };

    const mockGetAlertsWithExclusions = () => {
        // Rename the fixture file alerts/alertsWithWhitelistedDeployments.json
        // and add exclusions property as prerequisites to fix the test:
        // xit('should exclude the deployment'
        cy.intercept('GET', `${api.alerts.alerts}?query=*`, {
            fixture: 'alerts/alertsWithExclusionDeployment.json', // TODO rename
        }).as('alertsWithExclusions');
    };

    const mockResolveAlert = () => {
        cy.intercept('PATCH', api.alerts.resolveAlert, {
            body: {},
        }).as('resolveAlert');
    };

    const mockGetPolicy = () => {
        cy.route('GET', api.policies.policy, {
            body: {},
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

    it('should have runtime violation information in the Violations tab', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstTableRowLink).click();
        cy.wait('@alertById');
        cy.get(ViolationsPageSelectors.details.violationTab);
        // TODO Violation Events and so on
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
        mockGetAlertsWithExclusions();
        mockResolveAlert();
        mockGetPolicy();
        cy.get(ViolationsPageSelectors.lastTableRow).find('[type="checkbox"]').check();
        cy.get('.panel-actions button').first().click();
        cy.get('.ReactModal__Content .btn.btn-success').click();
        cy.wait('@resolveAlert');
        cy.wait('@getPolicy');
        cy.visit('/main/violations');
        cy.wait('@alertsWithExclusions');
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
            expect(message).to.contain('Runtime data was evaluated against this security policy');
        });
    });

    it('should have deployment information in the Deployment tab', () => {
        mockGetAlert();
        cy.get(ViolationsPageSelectors.firstTableRowLink).click();
        cy.wait('@alertById').then((interception) => {
            const { deployment } = interception.response.body;
            cy.intercept('GET', `${api.risks.getDeployment}`, {
                body: deployment,
            }).as('deployment');
            cy.get(ViolationsPageSelectors.details.deploymentTab).click();
            cy.wait('@deployment');

            cy.get(ViolationsPageSelectors.deployment.overview).should('exist');
            cy.get(ViolationsPageSelectors.deployment.containerConfiguration).should('exist');
            cy.get(ViolationsPageSelectors.deployment.securityContext).should('exist');
            cy.get(ViolationsPageSelectors.deployment.portConfiguration).should('exist');

            // TODO does not exist: should it?
            // cy.get(ViolationsPageSelectors.deployment.snapshotWarning).should('exist');
        });
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
    });

    it('should sort violations when clicking on a table header', () => {
        // First click sorts in descending order.
        cy.intercept(
            {
                method: 'GET',
                pathname: api.alerts.alerts,
                query: {
                    'pagination.sortOption.field': 'Policy',
                    'pagination.sortOption.reversed': 'false',
                },
            },
            {
                fixture: 'alerts/alerts.json',
            }
        ).as('alertsPolicyDescending');

        cy.get(ViolationsPageSelectors.table.column.policy).click();
        cy.wait('@alertsPolicyDescending').then((interception) => {
            cy.get(ViolationsPageSelectors.firstTableRow).should('contain', 'ip-masq-agent');

            const { alerts } = interception.response.body;
            cy.intercept(
                {
                    method: 'GET',
                    pathname: api.alerts.alerts,
                    query: {
                        'pagination.sortOption.field': 'Policy',
                        'pagination.sortOption.reversed': 'true',
                    },
                },
                {
                    body: {
                        alerts: alerts.sort(
                            (a, b) => -1 * a.policy.name.localeCompare(b.policy.name)
                        ),
                    },
                }
            ).as('alertsPolicyAscending');

            // Second click sorts in ascending order.
            cy.get(ViolationsPageSelectors.table.column.policy).click();
            cy.wait('@alertsPolicyAscending');
            cy.get(ViolationsPageSelectors.firstTableRow).should('contain', 'metadata-proxy-v0.1');
        });
    });
});
