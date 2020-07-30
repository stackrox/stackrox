import { url, selectors as dashboardSelectors } from '../constants/DashboardPage';
import { selectors as riskSelectors } from '../constants/RiskPage';
import selectors from '../selectors/index';
import withAuth from '../helpers/basicAuth';

describe('Dashboard Flow', () => {
    withAuth();

    it('should show high level stats', () => {
        cy.visit(url);
        cy.get(selectors.navigation.summaryCounts, { timeout: 7000 }).each(($el) => {
            cy.wrap($el).should('not.contain.text', '0');
        });
    });

    it('should validate violations stats/counts', () => {
        cy.visit(url);
        cy.get(dashboardSelectors.severityTiles, { timeout: 7000 }).each(($el) => {
            cy.wrap($el).should('not.contain.text', '0');
        });
    });

    describe('Drill down of top risky deployment', () => {
        const deploymentName = 'asset-cache';

        it(`should navigate user to Risk page with top risky deployment (${deploymentName}) selected`, () => {
            cy.visit(url);
            cy.get(dashboardSelectors.topRiskyDeployments).contains(deploymentName).click();
            cy.get(selectors.page.pageHeader).contains('Risk');
            cy.get(selectors.table.activeRow).contains(deploymentName);
            cy.get(selectors.panel.panelHeader).eq(1).contains(deploymentName);
        });

        it('should have /bin/bash command execution details', () => {
            cy.get(selectors.tab.tabs).contains('Process Discovery').click();
            cy.get(riskSelectors.suspiciousProcesses).contains('/bin/bash').click();
            cy.get(selectors.collapsible.card)
                .eq(0)
                .find(selectors.collapsible.body)
                .children()
                .should('not.be.empty');
        });

        it('should be able to navigate back to the Dashboard', () => {
            cy.get(selectors.navigation.leftNavBar).contains('Dashboard').click();
            cy.get(selectors.page.pageHeader).contains('Dashboard');
        });
    });

    describe('Drill down of Violation to DevOps Best Practices', () => {
        const policyCategory = 'DevOps Best Practice';
        const severity = 'MEDIUM_SEVERITY';
        const policyName = 'No Resource Request limits';

        beforeEach(() => {
            cy.server();
            cy.route('/v1/alertscount?query=Category:DevOps Best Practices').as(
                'getAlertsForDevOpsBestPractices'
            );
        });

        it('should navigate user to the Violations page with the filter "Category: Devops Best Practices", "Severity: Medium_Severity" added to the filter bar', () => {
            cy.visit(url);
            cy.get(`${dashboardSelectors.policyCategoryViolations}:contains(${policyCategory})`)
                .find(dashboardSelectors.chart.medSeveritySector)
                .eq(0)
                .click({ force: true });
            cy.get(selectors.search.chips).eq(0).contains('Category:');
            cy.get(selectors.search.chips).eq(1).contains(policyCategory);
            cy.get(selectors.search.chips).eq(2).contains('Severity:');
            cy.get(selectors.search.chips).eq(3).contains(severity);
        });

        it(`should show violation details for a violation that violates the "${policyName}" Policy`, () => {
            cy.get(selectors.table.rows).eq(0).click();
            cy.get(selectors.tab.activeTab).contains('Violation');
        });

        it('should remove Severity filter from search bar and still see violations for "Category: Devops Best Practices"', () => {
            cy.get(selectors.search.input).type('{backspace}');
            cy.get(selectors.search.input).type('{backspace}');
            cy.get(selectors.search.chips).eq(0).contains('Category:');
            cy.get(selectors.search.chips).eq(1).contains(policyCategory);
            cy.wait('@getAlertsForDevOpsBestPractices');
            cy.get(selectors.table.rows).each(($el, index) => {
                cy.get(`${selectors.table.rows}:nth(${index}) ${selectors.table.cells}:nth(7)`)
                    .invoke('text')
                    .then((text) => {
                        expect(text).to.match(/Multiple|DevOps Best Practices/);
                    });
            });
        });

        // @TODO modify test to test sorting once backend bug is fixed
        it('should sort violations list alphabetically by policy name', () => {
            cy.get(selectors.table.columnHeaders)
                .contains('Policy')
                .click({ force: true })
                .click({ force: true });
        });
    });
});
