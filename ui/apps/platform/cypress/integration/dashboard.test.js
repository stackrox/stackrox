import { url as dashboardUrl, selectors } from '../constants/DashboardPage';
import { url as riskUrl } from '../constants/RiskPage';

import {
    url as violationsUrl,
    selectors as violationsSelectors,
} from '../constants/ViolationsPage';
import {
    url as complianceUrl,
    selectors as complianceSelectors,
} from '../constants/CompliancePage';
import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';
import { visitMainDashboard } from '../helpers/main';
import baseSelectors from '../selectors/index';

describe('Dashboard page', () => {
    withAuth();

    it('should select item in nav bar', () => {
        cy.visit(dashboardUrl);
        cy.get(selectors.navLink).should('have.class', 'pf-m-current');
    });

    it('should display system violations tiles', () => {
        cy.intercept('GET', api.alerts.countsByCluster, {
            fixture: 'alerts/countsByCluster-single.json',
        });

        visitMainDashboard();

        cy.get(selectors.sectionHeaders.systemViolations).next('div').children().as('riskTiles');

        cy.get('@riskTiles').spread((aCritical, aHigh, aMedium, aLow) => {
            cy.wrap(aLow).should('have.text', '2Low');
            cy.wrap(aMedium).should('have.text', '1Medium');
            cy.wrap(aHigh).should('have.text', '0High');
            cy.wrap(aCritical).should('have.text', '0Critical');
        });
    });

    it('should not navigate to the violations page when clicking the critical severity risk tile', () => {
        visitMainDashboard();
        cy.get(selectors.sectionHeaders.systemViolations).next('div').children().as('riskTiles');

        cy.get('@riskTiles').first().click();
        cy.location().should((location) => {
            expect(location.pathname).to.eq(dashboardUrl);
        });
    });

    it('should navigate to violations page when clicking the low severity tile', () => {
        visitMainDashboard();

        // Click on the "Low" severity tile to link to the Violations page, and then ensure
        // the number of filtered Violations matches what was displayed on the Dashboard
        cy.get(`${selectors.sectionHeaders.systemViolations} + div *:contains("Low")`).then(
            ([lowSeverityTile]) => {
                const lowSeverityCount = lowSeverityTile.innerText.replace(/\D.*/, '');
                cy.wrap(lowSeverityTile).click();
                cy.location().should((location) => {
                    expect(location.pathname).to.eq(violationsUrl);
                    expect(location.search).to.eq('?search[Severity]=LOW_SEVERITY');
                });
                cy.get(violationsSelectors.resultsFoundHeader(lowSeverityCount));
            }
        );
    });

    it('should navigate to compliance standards page when clicking on standard', () => {
        visitMainDashboard();
        cy.get(selectors.sectionHeaders.compliance).should('exist');
        cy.get(selectors.chart.legendLink).click();
        cy.location().should((location) => {
            expect(location.href).to.include(complianceUrl.list.standards.CIS_Docker_v1_2_0);
        });
        cy.get(complianceSelectors.list.table.header).should('exist');
    });

    it('should display violations by cluster chart for single cluster', () => {
        cy.intercept('GET', api.alerts.countsByCluster, {
            fixture: 'alerts/countsByCluster-single.json',
        });

        visitMainDashboard();

        cy.get(selectors.sectionHeaders.violationsByClusters).next().as('chart');

        cy.get('@chart').within(() => {
            cy.get(selectors.chart.xAxis).should('contain', 'Kubernetes Cluster 0');
            cy.get(selectors.chart.grid).spread((grid) => {
                // from alerts fixture : low = 2, medium = 1, therefore medium's height should be twice less
                const { height } = grid.getBBox();
                cy.get(selectors.chart.lowSeverityBar).should('have.attr', 'height', `${height}`);
                cy.get(selectors.chart.medSeverityBar).should(
                    'have.attr',
                    'height',
                    `${height / 2}`
                );
            });
        });

        // TODO: validate clicking on any bar (for some reason '.click()' doesn't simply work for D3 chart)
    });

    it('should display violations by cluster chart for two clusters', () => {
        cy.intercept('GET', api.alerts.countsByCluster, {
            fixture: 'alerts/countsByCluster-couple.json',
        });

        visitMainDashboard();

        cy.get(selectors.sectionHeaders.violationsByClusters)
            .next()
            .find(selectors.chart.xAxis)
            .should('contain', 'Kubernetes Cluster 1');
    });

    it('should display events by time charts', () => {
        cy.intercept('GET', api.dashboard.timeseries, {
            fixture: 'alerts/alertsByTimeseries.json',
        });
        visitMainDashboard();
        cy.get(selectors.sectionHeaders.eventsByTime).next().find(selectors.timeseries);
    });

    it('should display violations category chart', () => {
        cy.intercept('GET', api.alerts.countsByCategory, {
            fixture: 'alerts/countsByCategory.json',
        });

        visitMainDashboard();

        cy.get(selectors.sectionHeaders.securityBestPractices).next().as('chart');
        cy.get('@chart').find(selectors.chart.legendItem).should('have.text', 'Low');

        // TODO: validate clicking on any sector (for some reason '.click()' isn't stable for D3 chart)
    });

    it('should display top risky deployments', () => {
        const { table } = baseSelectors;

        visitMainDashboard();

        // Gets the list of top risky deployments on the dashboard and checks to
        // see that the order matches on the Risk page.
        cy.get(`${selectors.sectionHeaders.topRiskyDeployments} + div li`).then(($deployments) => {
            cy.get(selectors.buttons.viewAll).click();
            cy.location('pathname').should('eq', riskUrl);

            $deployments.each((i, elem) => {
                const deploymentName = elem.innerText.replace(/\n.*/, '');
                const nthGroup = `${table.body} ${table.group}:nth-child(${i + 1})`;
                const firstCell = `${table.cells}:nth-child(1)`;
                cy.get(`${nthGroup} ${firstCell}:contains("${deploymentName}")`);
            });
        });
    });

    it('should display a search input with only the cluster search modifier', () => {
        cy.visit(dashboardUrl);
        cy.get(selectors.searchInput).type('Cluster:{enter}', { force: true });
        cy.get(selectors.searchInput).type('remote{enter}', { force: true });
    });

    it('should show the proper empty states', () => {
        cy.intercept('GET', api.alerts.countsByCategory, {
            body: { groups: [] },
        }).as('alertsByCategory');
        cy.intercept('GET', api.alerts.countsByCluster, {
            body: { groups: [] },
        }).as('alertsByCluster');

        visitMainDashboard();

        cy.get(selectors.sectionHeaders.securityBestPractices).should('not.exist');
        cy.get(selectors.sectionHeaders.devopsBestPractices).should('not.exist');

        cy.get(selectors.sectionHeaders.violationsByClusters)
            .next()
            .should(
                'have.text',
                'No data available. Please ensure your cluster is properly configured.'
            );
    });
});
