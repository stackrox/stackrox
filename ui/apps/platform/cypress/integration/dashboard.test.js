import { url as dashboardUrl, selectors } from '../constants/DashboardPage';
import { url as riskUrl } from '../constants/RiskPage';

import {
    url as violationsUrl,
    selectors as violationsSelectors,
} from '../constants/ViolationsPage';
import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';
import { hasFeatureFlag } from '../helpers/features';
import { visitMainDashboard, visitMainDashboardViaRedirectFromUrl } from '../helpers/main';
import baseSelectors from '../selectors/index';

// For future redesign of main dashboard, separate tests for these requests into a separate file.

const alertsSummaryCountsByCluster0 = { groups: [] };

const alertsSummaryCountsByCluster1 = {
    groups: [
        {
            group: 'Kubernetes Cluster 0',
            counts: [
                { severity: 'LOW_SEVERITY', count: '2' },
                { severity: 'MEDIUM_SEVERITY', count: '1' },
                { severity: 'CRITICAL_SEVERITY', count: '0' },
            ],
        },
    ],
};

const alertsSummaryCountsByCluster2 = {
    groups: [
        {
            group: 'Kubernetes Cluster 0',
            counts: [
                { severity: 'LOW_SEVERITY', count: '2' },
                { severity: 'MEDIUM_SEVERITY', count: '1' },
            ],
        },
        {
            group: 'Kubernetes Cluster 1',
            counts: [
                { severity: 'HIGH_SEVERITY', count: '10' },
                { severity: 'CRITICAL_SEVERITY', count: '5' },
            ],
        },
    ],
};

describe('Dashboard page', () => {
    before(function beforeHook() {
        if (hasFeatureFlag('ROX_SECURITY_METRICS_PHASE_ONE')) {
            this.skip();
        }
    });

    withAuth();

    // Skip tests for elements which seem less likely to have a counterpart in future design.

    it('should select item in left nav', () => {
        visitMainDashboard();

        cy.get(selectors.navLink).should('have.class', 'pf-m-current');
    });

    it('should render navbar with Dashboard selected', () => {
        visitMainDashboardViaRedirectFromUrl('/');

        cy.get(selectors.navLink).should('have.class', 'pf-m-current');
    });

    it('should have the summary counts in the top header', () => {
        visitMainDashboard();

        const { summaryCount: summaryCountSelector } = selectors;
        cy.get(`${summaryCountSelector}:nth-child(1):contains("Cluster")`);
        cy.get(`${summaryCountSelector}:nth-child(2):contains("Node")`);
        cy.get(`${summaryCountSelector}:nth-child(3):contains("Violation")`);
        cy.get(`${summaryCountSelector}:nth-child(4):contains("Deployment")`);
        cy.get(`${summaryCountSelector}:nth-child(5):contains("Image")`);
        cy.get(`${summaryCountSelector}:nth-child(6):contains("Secret")`);
    });

    it('should display system violations tiles', () => {
        cy.intercept('GET', api.alerts.countsByCluster, {
            body: alertsSummaryCountsByCluster1,
        }).as('getAlertsCountsByCluster');
        visitMainDashboard();
        cy.wait('@getAlertsCountsByCluster');

        cy.get(`${selectors.severityTile}:contains("2Low")`);
        cy.get(`${selectors.severityTile}:contains("1Medium")`);
        cy.get(`${selectors.severityTile}:contains("0High")`);
        cy.get(`${selectors.severityTile}:contains("0Critical")`);
    });

    it('should not navigate to the violations page when clicking the critical severity risk tile', () => {
        // For future design of main dashboard: Why not? A link is valid, even if no violations right now.

        cy.intercept('GET', api.alerts.countsByCluster, {
            body: alertsSummaryCountsByCluster1,
        }).as('getAlertsCountsByCluster');
        visitMainDashboard();
        cy.wait('@getAlertsCountsByCluster');

        cy.get(`${selectors.severityTile}:contains("Critical")`).click();
        cy.location('pathname').should('eq', dashboardUrl);
    });

    it('should navigate to violations page when clicking the low severity tile', () => {
        visitMainDashboard();

        // Click on the "Low" severity tile to link to the Violations page, and then ensure
        // the number of filtered Violations matches what was displayed on the Dashboard
        cy.get(`${selectors.severityTile}:contains("Low")`).then(([lowSeverityTile]) => {
            const lowSeverityCount = Number(lowSeverityTile.innerText.replace(/\D.*/, ''));

            cy.wrap(lowSeverityTile).click();

            cy.location('pathname').should('eq', violationsUrl);
            cy.location('search').should('eq', '?s[Severity]=LOW_SEVERITY');
            cy.get(violationsSelectors.resultsFoundHeader(lowSeverityCount));
        });
    });

    it('should display violations by cluster chart for single cluster', () => {
        cy.intercept('GET', api.alerts.countsByCluster, {
            body: alertsSummaryCountsByCluster1,
        }).as('getAlertsCountsByCluster');
        visitMainDashboard();
        cy.wait('@getAlertsCountsByCluster');

        cy.get(selectors.chart.xAxis).should('contain', 'Kubernetes Cluster 0');

        // For future design of main dashboard: Accessible data does not need tricky assertions.
        cy.get(selectors.chart.grid).then(([grid]) => {
            // from alerts fixture : low = 2, medium = 1, therefore medium's height should be twice less
            const { height } = grid.getBBox();
            cy.get(selectors.chart.lowSeverityBar).should('have.attr', 'height', `${height}`);
            cy.get(selectors.chart.medSeverityBar).should('have.attr', 'height', `${height / 2}`);
        });

        // TODO: validate clicking on any bar (for some reason '.click()' doesn't simply work for D3 chart)
    });

    it('should display violations by cluster chart for two clusters', () => {
        cy.intercept('GET', api.alerts.countsByCluster, {
            body: alertsSummaryCountsByCluster2,
        }).as('getAlertsCountsByCluster');
        visitMainDashboard();
        cy.wait('@getAlertsCountsByCluster');

        cy.get(selectors.chart.xAxis).should('contain', 'Kubernetes Cluster 1');
    });

    it.skip('should display events by time charts', () => {
        cy.intercept('GET', api.dashboard.timeseries, {
            fixture: 'alerts/alertsByTimeseries.json',
        });
        visitMainDashboard();
        cy.get(selectors.sectionHeaders.eventsByTime).next().find(selectors.timeseries);
    });

    it.skip('should display violations category chart', () => {
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
            cy.intercept('GET', api.risks.riskyDeployments).as('riskyDeployments');
            cy.get(selectors.buttons.viewAll).click();
            cy.wait('@riskyDeployments');
            cy.get('h1:contains("Risk")');
            cy.location('pathname').should('eq', riskUrl);

            $deployments.each((i, elem) => {
                const deploymentName = elem.innerText.replace(/\n.*/, '');
                const nthGroup = `${table.body} ${table.group}:nth-child(${i + 1})`;
                const firstCell = `${table.cells}:nth-child(1)`;
                cy.get(`${nthGroup} ${firstCell}:contains("${deploymentName}")`);
            });
        });
    });

    it.skip('should display a search input with only the cluster search modifier', () => {
        cy.visit(dashboardUrl);
        cy.get(selectors.searchInput).type('Cluster:{enter}', { force: true });
        cy.get(selectors.searchInput).type('remote{enter}', { force: true });
    });

    it('should show the proper empty states', () => {
        cy.intercept('GET', api.alerts.countsByCluster, {
            body: alertsSummaryCountsByCluster0,
        }).as('getAlertsCountsByCluster');
        visitMainDashboard();
        cy.wait('@getAlertsCountsByCluster');

        cy.get(selectors.chart.resultsMessage).should(
            'have.text',
            'No data available. Please ensure your cluster is properly configured.'
        );
    });
});
