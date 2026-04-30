import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
import pf6 from '../../selectors/pf6';

import {
    clickTab,
    viewFirstRiskDeployment,
    viewRiskDeploymentInNetworkGraph,
    visitRiskDeployments,
    visitRiskDeploymentsWithSearchQuery,
} from './Risk.helpers';
import { selectors as RiskPageSelectors } from './Risk.selectors';

describe('Risk', () => {
    withAuth();

    describe('without mock API', () => {
        it('should have selected item in nav bar', () => {
            visitRiskDeployments('Platform view');

            cy.get(RiskPageSelectors.risk).should('have.class', 'pf-m-current');
        });

        it('should have title and table column headings', () => {
            visitRiskDeployments('Platform view');

            cy.title().should('match', getRegExpForTitleWithBranding('Risk'));

            cy.get('th:contains("Name")');
            cy.get('th:contains("Created")');
            cy.get('th:contains("Cluster")');
            cy.get('th:contains("Namespace")');
            cy.get('th:contains("Priority")');
        });

        it('should open detail page for deployment', () => {
            visitRiskDeployments('Platform view');
            viewFirstRiskDeployment();
        });

        // TODO add relevant tests for error messages in PatternFly

        it('should open the detail page to view risk indicators, deployment details, and process discovery tabs', () => {
            visitRiskDeployments('Platform view');
            viewFirstRiskDeployment();

            cy.get('[role="tab"]:contains("Risk indicators")');
            cy.get('[role="tab"]:contains("Deployment details")');
            cy.get('[role="tab"]:contains("Process discovery")');
        });

        it('should navigate from Risk Page to Vulnerability Management Image Page', () => {
            visitRiskDeployments('Platform view');
            viewFirstRiskDeployment();

            clickTab('Deployment details');

            cy.contains('h3', 'Container configuration')
                .parents(pf6.card)
                .first()
                .within(() => {
                    cy.get('button[aria-expanded="false"]').first().click();
                    cy.get('a').first().click();
                });

            cy.location('pathname')
                .should('contain', '/main/vulnerabilities/')
                .and('contain', '/image');
        });
    });

    describe('with actual API', () => {
        it('should navigate to network page with selected deployment', () => {
            visitRiskDeployments('Platform view');
            viewFirstRiskDeployment().then((deploymentName) => {
                viewRiskDeploymentInNetworkGraph(deploymentName);
            });
        });

        const searchPlaceholderSelector = `${RiskPageSelectors.search.valueContainer} input[placeholder="Filter deployments"]`;

        it('should not have anything in search bar when URL has no search params', () => {
            visitRiskDeployments('Platform view');

            // Positive assertion:
            cy.get(searchPlaceholderSelector);
            // Negative assertion:
            cy.get(RiskPageSelectors.search.searchLabels).should('not.exist');
        });

        it('should have a single URL search param key/value pair in its search bar', () => {
            const nsOption = 'Namespace';
            const nsValue = 'stackrox';

            visitRiskDeploymentsWithSearchQuery('Platform view', `s[${nsOption}]=${nsValue}`);

            cy.get(searchPlaceholderSelector).should('not.exist');
            cy.get(RiskPageSelectors.search.searchLabels).should('have.length', 2);
            cy.get(`${RiskPageSelectors.search.searchLabels}:nth(0)`).should(
                'have.text',
                `${nsOption}:`
            );
            cy.get(`${RiskPageSelectors.search.searchLabels}:nth(1)`).should('have.text', nsValue);
        });

        it('should have multiple URL search param key/value pairs in its search bar', () => {
            const nsOption = 'Namespace';
            const nsValue = 'stackrox';
            const deployOption = 'Deployment';
            const deployValue = 'scanner';

            visitRiskDeploymentsWithSearchQuery(
                'Platform view',
                `s[${nsOption}]=${nsValue}&s[${deployOption}]=${deployValue}`
            );

            cy.get(searchPlaceholderSelector).should('not.exist');
            cy.get(RiskPageSelectors.search.searchLabels).should('have.length', 4);
            cy.get(`${RiskPageSelectors.search.searchLabels}:nth(0)`).should(
                'have.text',
                `${nsOption}:`
            );
            cy.get(`${RiskPageSelectors.search.searchLabels}:nth(1)`).should('have.text', nsValue);
            cy.get(`${RiskPageSelectors.search.searchLabels}:nth(2)`).should(
                'have.text',
                `${deployOption}:`
            );
            cy.get(`${RiskPageSelectors.search.searchLabels}:nth(3)`).should(
                'have.text',
                deployValue
            );
        });
    });
});
