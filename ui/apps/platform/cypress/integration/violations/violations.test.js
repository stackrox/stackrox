import { selectors } from '../../constants/ViolationsPage';
import { selectors as PoliciesPageSelectors } from '../../constants/PoliciesPage';
import withAuth from '../../helpers/basicAuth';
import {
    clickDeploymentTabWithFixture,
    sortViolationsTableByColumn,
    visitViolationFromTableWithFixture,
    visitViolationWithFixture,
    visitViolations,
    visitViolationsFromLeftNav,
    visitViolationsWithFixture,
} from '../../helpers/violations';

describe('Violations page', () => {
    withAuth();

    it('should visit via left nav', () => {
        visitViolationsFromLeftNav();
    });

    it('should select item in left nav', () => {
        visitViolations();

        cy.get(selectors.navLink).should('have.class', 'pf-m-current');
    });

    it('should have violations in table', () => {
        visitViolationsWithFixture('alerts/alerts.json');

        const count = 2;
        cy.get(selectors.resultsFoundHeader(count));
        cy.get(selectors.table.rows).should('have.length', count);
    });

    it('should have columns in table', () => {
        visitViolationsWithFixture('alerts/alerts.json');

        cy.get('th[scope="col"]:contains("Policy")');
        cy.get('th[scope="col"]:contains("Entity")');
        cy.get('th[scope="col"]:contains("Type")');
        cy.get('th[scope="col"]:contains("Enforced")');
        cy.get('th[scope="col"]:contains("Severity")');
        cy.get('th[scope="col"]:contains("Categories")');
        cy.get('th[scope="col"]:contains("Lifecycle")');
        cy.get('th[scope="col"]:contains("Time")');

        cy.get(`${selectors.firstTableRow} td[data-label="Entity"]`).should(
            'contain',
            'ip-masq-agent'
        ); // table cell also has cluster/namespace
        cy.get(`${selectors.firstTableRow} td[data-label="Type"]`).should(
            'have.text',
            'deployment'
        );
        cy.get(`${selectors.firstTableRow} td[data-label="Lifecycle"]`).should(
            'have.text',
            'Runtime'
        );
    });

    it('should go to the detail page on row click', () => {
        visitViolationsWithFixture('alerts/alerts.json');
        visitViolationFromTableWithFixture('alerts/alertFirstInAlerts.json');

        cy.get(selectors.details.page);
        cy.get(selectors.details.title).should('have.text', 'Misuse of iptables');
        cy.get(selectors.details.subtitle).should('have.text', 'in "ip-masq-agent" deployment');
    });

    it('should have 4 tabs in the sidepanel', () => {
        visitViolationWithFixture('alerts/alertFirstInAlerts.json');

        cy.get(selectors.details.tabs).should('have.length', 4);
        cy.get(selectors.details.violationTab);
        cy.get(selectors.details.enforcementTab);
        cy.get(selectors.details.deploymentTab);
        cy.get(selectors.details.policyTab);
    });

    it('should have runtime violation information in the Violations tab', () => {
        visitViolationWithFixture('alerts/alertFirstInAlerts.json');

        cy.get(selectors.details.violationTab);
        // TODO Violation Events and so on
    });

    it('should contain correct action buttons for the lifecycle stage', () => {
        visitViolationsWithFixture('alerts/alerts.json');

        // Lifecycle: Runtime
        cy.get(`${selectors.firstTableRow} ${selectors.actions.btn}`).click(); // click kabob to open actions menu
        cy.get(selectors.firstTableRow)
            .get(selectors.actions.excludeDeploymentBtn)
            .should('exist')
            .get(selectors.actions.resolveBtn)
            .should('exist')
            .get(selectors.actions.resolveAndAddToBaselineBtn)
            .should('exist');
        cy.get(`${selectors.firstTableRow} ${selectors.actions.btn}`).click(); // click kabob to close actions menu

        // Lifecycle: Deploy
        cy.get(`${selectors.lastTableRow} ${selectors.actions.btn}`).click(); // click kabob to open actions menu
        cy.get(selectors.lastTableRow)
            .get(selectors.actions.resolveBtn)
            .should('not.exist')
            .get(selectors.actions.resolveAndAddToBaselineBtn)
            .should('not.exist')
            .get(selectors.actions.excludeDeploymentBtn)
            .should('exist');
        cy.get(`${selectors.lastTableRow} ${selectors.actions.btn}`).click(); // click kabob to close actions menu
    });

    // TODO test of bulk actions

    // TODO mock no-op request for any action which would prevent repeatable test runs in local deployment

    it('should have enforcement information in the Enforcement tab', () => {
        visitViolationWithFixture('alerts/alertFirstInAlerts.json');

        cy.get(selectors.details.enforcementTab).click();
        cy.get(selectors.enforcement.detailMessage).should('contain', 'Kill Pod');
        cy.get(selectors.enforcement.explanationMessage).should(
            'contain',
            'Runtime data was evaluated against this security policy'
        );
    });

    it('should have deployment information in the Deployment tab', () => {
        visitViolationWithFixture('alerts/alertFirstInAlerts.json');
        clickDeploymentTabWithFixture('alerts/deploymentForAlertFirstInAlerts.json');

        cy.get(selectors.deployment.overview);
        cy.get(selectors.deployment.containerConfiguration);
        cy.get(`${selectors.deployment.containerConfiguration} [data-testid="commands"]`).should(
            'not.exist'
        );
        cy.get(selectors.deployment.securityContext);
        cy.get(selectors.deployment.portConfiguration);
    });

    it('should show deployment information in the Deployment Details tab with no container configuration values', () => {
        visitViolationWithFixture('alerts/alertWithEmptyContainerConfig.json');
        clickDeploymentTabWithFixture('alerts/deploymentWithEmptyContainerConfig.json');

        cy.get(selectors.deployment.containerConfiguration);
        // TODO need more positive and negative assertions to contrast deployments in this and the previous test.
        cy.get(`${selectors.deployment.containerConfiguration} [data-testid="commands"]`).should(
            'not.exist'
        );
    });

    it('should have policy information in the Policy Details tab', () => {
        visitViolationWithFixture('alerts/alertFirstInAlerts.json');

        cy.get(selectors.details.policyTab).click();
        cy.get(PoliciesPageSelectors.policyDetailsPanel.detailsSection);
    });

    it('should sort violations when clicking on a table header', () => {
        visitViolations();

        // First click sorts in descending order.
        sortViolationsTableByColumn('Policy');

        cy.get('td[data-label="Policy"] a').should(($anchors) => {
            const firstName = $anchors.first().text();
            const lastName = $anchors.last().text();
            const policyNamesReceivedOrder = [firstName, lastName];
            const policyNamesExpectedOrder =
                firstName.localeCompare(lastName) >= 0
                    ? [firstName, lastName]
                    : [lastName, firstName];
            expect(policyNamesReceivedOrder).to.deep.equal(policyNamesExpectedOrder);
        });

        // Second click sorts in ascending order.
        sortViolationsTableByColumn('Policy');

        cy.get('td[data-label="Policy"] a').should(($anchors) => {
            const firstName = $anchors.first().text();
            const lastName = $anchors.last().text();
            const policyNamesReceivedOrder = [firstName, lastName];
            const policyNamesExpectedOrder =
                firstName.localeCompare(lastName) <= 0
                    ? [firstName, lastName]
                    : [lastName, firstName];
            expect(policyNamesReceivedOrder).to.deep.equal(policyNamesExpectedOrder);
        });
    });
});
