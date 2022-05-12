import { selectors as ViolationsPageSelectors } from '../../constants/ViolationsPage';
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

        cy.get(ViolationsPageSelectors.navLink).should('have.class', 'pf-m-current');
    });

    it('should have violations in table', () => {
        visitViolationsWithFixture('alerts/alerts.json');

        const count = 2;
        cy.get(ViolationsPageSelectors.resultsFoundHeader(count));
        cy.get(ViolationsPageSelectors.table.rows).should('have.length', count);
    });

    it('should have Lifecycle column in table', () => {
        visitViolationsWithFixture('alerts/alerts.json');

        cy.get(ViolationsPageSelectors.table.column.lifecycle);
        cy.get(`${ViolationsPageSelectors.firstTableRow} td[data-label="Lifecycle"]`).should(
            'have.text',
            'Runtime'
        );
    });

    it('should go to the detail page on row click', () => {
        visitViolationsWithFixture('alerts/alerts.json');
        visitViolationFromTableWithFixture('alerts/alert0.json');

        cy.get(ViolationsPageSelectors.details.page);
        cy.get(ViolationsPageSelectors.details.title).should('have.text', 'Misuse of iptables');
        cy.get(ViolationsPageSelectors.details.subtitle).should(
            'have.text',
            'in "ip-masq-agent" deployment'
        );
    });

    it('should have Entity column in table', () => {
        visitViolationsWithFixture('alerts/alerts.json');

        cy.get(ViolationsPageSelectors.table.column.entity);
        cy.get(`${ViolationsPageSelectors.firstTableRow} td[data-label="Entity"]`).should(
            'contain',
            'ip-masq-agent'
        );
        // Table cell also has cluster/namespace.
    });

    it('should have Type column in table', () => {
        visitViolationsWithFixture('alerts/alerts.json');

        cy.get(ViolationsPageSelectors.table.column.type);
        cy.get(`${ViolationsPageSelectors.firstTableRow} td[data-label="Type"]`).should(
            'have.text',
            'deployment'
        );
    });

    it('should have 4 tabs in the sidepanel', () => {
        visitViolationWithFixture('alerts/alert0.json');

        cy.get(ViolationsPageSelectors.details.tabs).should('have.length', 4);
        cy.get(ViolationsPageSelectors.details.violationTab).should('exist');
        cy.get(ViolationsPageSelectors.details.enforcementTab).should('exist');
        cy.get(ViolationsPageSelectors.details.deploymentTab).should('exist');
        cy.get(ViolationsPageSelectors.details.policyTab).should('exist');
    });

    it('should have runtime violation information in the Violations tab', () => {
        visitViolationWithFixture('alerts/alert0.json');

        cy.get(ViolationsPageSelectors.details.violationTab);
        // TODO Violation Events and so on
    });

    it('should contain correct action buttons for the lifecycle stage', () => {
        visitViolationsWithFixture('alerts/alerts.json');

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

    // TODO test of bulk actions

    // TODO mock no-op request for any action which would prevent repeatable test runs in local deployment

    it('should have enforcement information in the Enforcement tab', () => {
        visitViolationWithFixture('alerts/alert0.json');

        cy.get(ViolationsPageSelectors.details.enforcementTab).click();
        cy.get(ViolationsPageSelectors.enforcement.detailMessage).should('contain', 'Kill Pod');
        cy.get(ViolationsPageSelectors.enforcement.explanationMessage).should(
            'contain',
            'Runtime data was evaluated against this security policy'
        );
    });

    it('should have deployment information in the Deployment tab', () => {
        visitViolationWithFixture('alerts/alert0.json');
        clickDeploymentTabWithFixture('alerts/deployment0.json');

        cy.get(ViolationsPageSelectors.deployment.overview);
        cy.get(ViolationsPageSelectors.deployment.containerConfiguration);
        cy.get(
            `${ViolationsPageSelectors.deployment.containerConfiguration} [data-testid="commands"]`
        ).should('not.exist');
        cy.get(ViolationsPageSelectors.deployment.securityContext);
        cy.get(ViolationsPageSelectors.deployment.portConfiguration);
    });

    it('should show deployment information in the Deployment Details tab with no container configuration values', () => {
        visitViolationWithFixture('alerts/alertWithEmptyContainerConfig.json');
        clickDeploymentTabWithFixture('alerts/deploymentWithEmptyContainerConfig.json');

        cy.get(ViolationsPageSelectors.deployment.containerConfiguration);
        // TODO need more positive and negative assertions to contrast deployments in this and the previous test.
        cy.get(
            `${ViolationsPageSelectors.deployment.containerConfiguration} [data-testid="commands"]`
        ).should('not.exist');
    });

    it('should have policy information in the Policy Details tab', () => {
        visitViolationWithFixture('alerts/alert0.json');

        cy.get(ViolationsPageSelectors.details.policyTab).click();
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
