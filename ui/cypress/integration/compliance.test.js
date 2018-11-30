import { selectors, url as complianceUrl } from './constants/CompliancePage';
import * as api from './constants/apiEndpoints';
import withAuth from './helpers/basicAuth';

describe('Compliance page', () => {
    withAuth();

    const setupMultipleClustersFixture = () => {
        cy.server();
        cy.fixture('clusters/couple.json').as('coupleCluster');
        cy.route('GET', api.clusters.list, '@coupleCluster').as('clusters');
        cy.visit('/');
        cy.get(selectors.compliance).click();
        cy.wait(['@clusters']);
    };
    const setupSingleClusterFixtures = () => {
        cy.server();
        cy.fixture('clusters/single.json').as('singleCluster');
        cy.route('GET', api.clusters.list, '@singleCluster').as('clusters');
        cy.fixture('benchmarks/configs.json').as('configs');
        cy.route('GET', api.benchmarks.configs, '@configs').as('benchConfigs');
        cy.fixture('benchmarks/dockerBenchScans.json').as('dockerBenchScans');
        cy.route('GET', api.benchmarks.benchmarkScans, '@dockerBenchScans').as('scanMetadata');
        cy.fixture('benchmarks/dockerBenchScan1.json').as('dockerBenchScan1');
        cy.route('GET', api.benchmarks.scans, '@dockerBenchScan1').as('benchScan');
        cy.visit(complianceUrl);
        cy.wait(['@clusters', '@benchConfigs', '@scanMetadata', '@benchScan']);
    };

    const loadCompliancePage = () => {
        cy.server();
        cy.route(api.clusters.list).as('clusters');
        cy.route(api.benchmarks.configs).as('benchConfigs');
        cy.route(api.benchmarks.benchmarkScans).as('benchScans');
        cy.visit(complianceUrl);

        // wait for all the data to come back, otherwise re-rendering can lead to detached elements,
        // see https://docs.cypress.io/guides/references/error-messages.html#cy-failed-because-the-element-you-are-chaining-off-of-has-become-detached-or-removed-from-the-dom
        cy.wait(['@clusters', '@benchConfigs', '@benchScans']);
    };

    it('should allow to set schedule', () => {
        cy.server();
        cy.route(api.benchmarks.schedules).as('setSchedule');
        loadCompliancePage();

        cy.get('select:first').select('Friday', { force: true });
        cy.get('select:last').select('05:00 PM', { force: true });
        cy.wait('@setSchedule');
        cy.reload(); // retrieve data from the server
        cy.get('select:first').should('have.value', 'Friday');
        cy.get('select:last').should('have.value', '05:00 PM');

        // update schedule
        cy.get('select:last').select('06:00 PM', { force: true });
        cy.wait('@setSchedule');
        cy.reload();
        cy.get('select:last').should('have.value', '06:00 PM');

        // remove schedule
        cy.get('select:first').select('None', { force: true });
        cy.get('select:last').should('have.value', null);
    });

    it('should have selected first cluster in Compliance nav bar', () => {
        setupMultipleClustersFixture();
        cy.get(selectors.firstNavLink).click({ force: true });
        cy.get(selectors.compliance).should('have.class', 'bg-primary-700');
        cy.url().should('contain', '/main/compliance/swarmCluster1');
        // first tab selected by default
        cy.get(selectors.benchmarkTabs)
            .first()
            .should('have.class', 'tab-active');
        cy.get(selectors.benchmarkTabs).should('contain', 'CIS Swarm v1.1.0 Benchmark');
    });

    it('should have selected second cluster in Compliance nav bar', () => {
        setupMultipleClustersFixture();
        cy.get(selectors.secondNavLink).click({ force: true });
        cy.url().should('contain', '/main/compliance/kubeCluster1');
        cy.get(selectors.benchmarkTabs).should('contain', 'CIS Kubernetes v1.2.0 Benchmark');
    });

    it('should allow scanning initiation', () => {
        cy.server();
        cy.route('POST', api.benchmarks.triggers, {}).as('trigger');
        loadCompliancePage();
        cy.get(selectors.scanNowButton).as('scanNow');

        cy.get('@scanNow').should('contain', 'Scan now');
        cy.get('@scanNow').click();
        cy.wait('@trigger');
        cy.get('@scanNow').should('not.contain', 'Scan now'); // spinner
    });

    it('should show scan results', () => {
        setupSingleClusterFixtures();
        cy.get(selectors.benchmarkTabs)
            .first()
            .should('contain', 'CIS Docker v1.1.0 Benchmark');
        cy.get(selectors.checkRows).should('have.length', 5);
        cy.get(selectors.passColumns)
            .last()
            .should('have.text', '0');
    });

    it('should show benchmark host results', () => {
        setupSingleClusterFixtures();
        cy.route(
            'GET',
            api.benchmarks.scanHostResults,
            'fx:benchmarks/dockerBenchmarkHostResults.json'
        ).as('dockerBenchmarkHostResults');

        cy.get(selectors.passColumns)
            .first()
            .click();
        cy.wait('@dockerBenchmarkHostResults');
        cy.get(selectors.hostColumns)
            .should('have.length', 1)
            .and('contain', 'linuxkit-025000000001');
    });
});
