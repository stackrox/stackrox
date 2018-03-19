import { url as dashboardUrl, selectors } from './pages/DashboardPage';
import { url as complianceUrl } from './pages/CompliancePage';
import { url as violationsUrl } from './pages/ViolationsPage';
import * as api from './apiEndpoints';

describe('Dashboard page', () => {
    it('should select item in nav bar', () => {
        cy.visit(dashboardUrl);
        cy.get(selectors.navLink).should('have.class', 'bg-primary-600');
    });

    it('should display environment risk tiles', () => {
        cy.server();
        cy.fixture('alerts/countsByCluster-single.json').as('countsByCluster');
        cy.route('GET', api.alerts.countsByCluster, '@countsByCluster').as('alertsByCluster');

        cy.visit(dashboardUrl);
        cy.wait('@alertsByCluster');

        cy
            .get(selectors.sectionHeaders.environmentRisk)
            .next('div')
            .children()
            .as('riskTiles');

        cy.get('@riskTiles').spread((aCritical, aHigh, aMedium, aLow) => {
            cy.wrap(aLow).should('have.text', '2Low');
            cy.wrap(aMedium).should('have.text', '1Medium');
            cy.wrap(aHigh).should('have.text', '0High');
            cy.wrap(aCritical).should('have.text', '0Critical');
        });

        // check clicking Low tile
        cy
            .get('@riskTiles')
            .last()
            .click();
        cy.location().should(location => {
            expect(location.pathname).to.eq(violationsUrl);
            expect(location.search).to.eq('?severity=LOW_SEVERITY');
        });
    });

    it('should display benchmarks data', () => {
        cy.server();
        cy.fixture('benchmarks/configs.json').as('configs');
        cy.route('GET', api.benchmarks.configs, '@configs').as('benchConfigs');
        cy.fixture('benchmarks/dockerBenchScans.json').as('dockerBenchScans');
        cy.route('GET', api.benchmarks.cisDockerScans, '@dockerBenchScans').as('scansMetadata');
        cy.fixture('benchmarks/dockerBenchScan1.json').as('dockerBenchScan1');
        cy.route('GET', api.benchmarks.scans, '@dockerBenchScan1').as('benchScan');

        cy.visit(dashboardUrl);
        cy.wait(['@benchConfigs', '@scansMetadata', '@benchScan']);

        cy
            .get(selectors.sectionHeaders.benchmarks)
            .next()
            .children()
            .as('benchmarkSummaries');
        cy
            .get('@benchmarkSummaries')
            .find('a')
            .first()
            .should('have.text', 'CIS Docker v1.1.0 Benchmark');

        cy
            .get('@benchmarkSummaries')
            .find('a')
            .next()
            .children()
            .spread((pass, warn, info, note) => {
                expect(pass.getAttribute('style')).to.have.string('width: 20%');
                expect(warn.getAttribute('style')).to.have.string('width: 40%');
                expect(info.getAttribute('style')).to.have.string('width: 20%');
                expect(note.getAttribute('style')).to.have.string('width: 20%');
            });
        cy
            .get('@benchmarkSummaries')
            .find('a')
            .first()
            .click();
        cy.location().should(location => {
            expect(location.pathname).to.eq(complianceUrl);
        });
    });

    it('should display violations by cluster chart for single cluster', () => {
        cy.server();
        cy.fixture('alerts/countsByCluster-single.json').as('countsByCluster');
        cy.route('GET', api.alerts.countsByCluster, '@countsByCluster').as('alertsByCluster');

        cy.visit(dashboardUrl);
        cy.wait('@alertsByCluster');

        cy
            .get(selectors.sectionHeaders.violationsByClusters)
            .next()
            .as('chart');

        cy.get('@chart').within(() => {
            cy.get(selectors.chart.xAxis).should('contain', 'Swarm Cluster 1');
            cy.get(selectors.chart.grid).spread(grid => {
                // from alerts fixture : low = 2, medium = 1, therefore medium's height should be twice less
                const { height } = grid.getBBox();
                cy.get(selectors.chart.lowSeverityBar).should('have.attr', 'height', `${height}`);
                cy
                    .get(selectors.chart.medSeverityBar)
                    .should('have.attr', 'height', `${height / 2}`);
            });
        });

        // TODO: validate clicking on any bar (for some reason '.click()' doesn't simply work for D3 chart)
    });

    it('should display violations by cluster chart for two clusters', () => {
        cy.server();
        cy.fixture('alerts/countsByCluster-couple.json').as('countsByCluster');
        cy.route('GET', api.alerts.countsByCluster, '@countsByCluster').as('alertsByCluster');

        cy.visit(dashboardUrl);
        cy.wait('@alertsByCluster');

        cy
            .get(selectors.sectionHeaders.violationsByClusters)
            .next()
            .find(selectors.chart.xAxis)
            .should('contain', 'Swarm Cluster 1')
            .and('contain', 'Kubernetes Cluster 1');
    });

    it('should display events by time charts', () => {
        cy.visit(dashboardUrl);
        cy
            .get(selectors.sectionHeaders.eventsByTime)
            .next()
            .find('svg.recharts-surface')
            .should('not.have.length', 0);
    });

    it('should display violations category chart', () => {
        cy.server();
        cy.fixture('alerts/countsByCategory.json').as('countsByCategory');
        cy.route('GET', api.alerts.countsByCategory, '@countsByCategory').as('alertsByCategory');

        cy.visit(dashboardUrl);
        cy.wait('@alertsByCategory');

        cy
            .get(selectors.sectionHeaders.containerConfiguration)
            .next()
            .as('chart');
        cy
            .get('@chart')
            .find(selectors.chart.legendItem)
            .should('have.text', 'Medium');

        // TODO: validate clicking on any sector (for some reason '.click()' isn't stable for D3 chart)
    });
});
