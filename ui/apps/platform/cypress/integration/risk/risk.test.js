import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    clickTab,
    viewRiskDeploymentByName,
    viewRiskDeploymentInNetworkGraph,
    visitRiskDeployments,
    visitRiskDeploymentsWithSearchQuery,
} from './Risk.helpers';
import { selectors as RiskPageSelectors } from './Risk.selectors';

describe('Risk', () => {
    withAuth();

    describe('without mock API', () => {
        it('should have selected item in nav bar', () => {
            visitRiskDeployments();

            cy.get(RiskPageSelectors.risk).should('have.class', 'pf-m-current');
        });

        it('should have title and table column headings', () => {
            visitRiskDeployments();

            cy.title().should('match', getRegExpForTitleWithBranding('Risk'));

            cy.get('th:contains("Name")');
            cy.get('th:contains("Created")');
            cy.get('th:contains("Cluster")');
            cy.get('th:contains("Namespace")');
            cy.get('th:contains("Priority")');
        });

        it('should open detail page for deployment', () => {
            visitRiskDeployments();
            viewRiskDeploymentByName('collector');
        });

        // TODO add relevant tests for error messages in PatternFly

        it('should open the detail page to view risk indicators, deployment details, and process discovery tabs', () => {
            visitRiskDeployments();
            viewRiskDeploymentByName('collector');

            cy.get('[role="tab"]:contains("Risk indicators")');
            cy.get('[role="tab"]:contains("Deployment details")');
            cy.get('[role="tab"]:contains("Process discovery")');
        });

        it('should navigate from Risk Page to Vulnerability Management Image Page', () => {
            visitRiskDeployments();
            viewRiskDeploymentByName('collector');

            clickTab('Deployment details');
            cy.get(RiskPageSelectors.imageLink).first().click();

            cy.location('pathname').should('contain', '/main/vulnerabilities/platform/image');
        });
    });

    describe('with actual API', () => {
        it('should navigate to network page with selected deployment', () => {
            visitRiskDeployments();
            viewRiskDeploymentByName('collector');
            viewRiskDeploymentInNetworkGraph('collector');
        });

        const searchPlaceholderSelector = `${RiskPageSelectors.search.valueContainer} input[placeholder="Filter deployments"]`;

        it('should not have anything in search bar when URL has no search params', () => {
            visitRiskDeployments();

            // Positive assertion:
            cy.get(searchPlaceholderSelector);
            // Negative assertion:
            cy.get(RiskPageSelectors.search.searchLabels).should('not.exist');
        });

        it('should have a single URL search param key/value pair in its search bar', () => {
            visitRiskDeployments();

            const nsOption = 'Namespace';
            const nsValue = 'stackrox';
            cy.get(
                `${RiskPageSelectors.table.dataRows} td[data-label="Namespace"]:contains("${nsValue}")`
            ).then((stackroxDeps) => {
                const stackroxCount = stackroxDeps.length;

                visitRiskDeploymentsWithSearchQuery(`?s[${nsOption}]=${nsValue}`);

                // Negative assertion:
                cy.get(searchPlaceholderSelector).should('not.exist');
                // Positive assertions:
                cy.get(RiskPageSelectors.search.searchLabels).should('have.length', 2);
                cy.get(`${RiskPageSelectors.search.searchLabels}:nth(0)`).should(
                    'have.text',
                    `${nsOption}:`
                );
                cy.get(`${RiskPageSelectors.search.searchLabels}:nth(1)`).should(
                    'have.text',
                    nsValue
                );

                cy.get(RiskPageSelectors.table.dataRows).should('have.length', stackroxCount);
            });
        });

        it('should have multiple URL search param key/value pairs in its search bar', () => {
            visitRiskDeployments();

            const nsOption = 'Namespace';
            const nsValue = 'stackrox';
            const deployOption = 'Deployment';
            const deployValue = 'scanner';
            cy.get(
                `${RiskPageSelectors.table.dataRows} td[data-label="Name"]:contains("${deployValue}")`
            ).then((staticDeps) => {
                const staticCount = staticDeps.length;

                visitRiskDeploymentsWithSearchQuery(
                    `?s[${nsOption}]=${nsValue}&s[${deployOption}]=${deployValue}`
                );

                // Negative assertion:
                cy.get(searchPlaceholderSelector).should('not.exist');
                // Positive assertions:
                cy.get(RiskPageSelectors.search.searchLabels).should('have.length', 4);
                cy.get(`${RiskPageSelectors.search.searchLabels}:nth(0)`).should(
                    'have.text',
                    `${nsOption}:`
                );
                cy.get(`${RiskPageSelectors.search.searchLabels}:nth(1)`).should(
                    'have.text',
                    nsValue
                );
                cy.get(`${RiskPageSelectors.search.searchLabels}:nth(2)`).should(
                    'have.text',
                    `${deployOption}:`
                );
                cy.get(`${RiskPageSelectors.search.searchLabels}:nth(3)`).should(
                    'have.text',
                    deployValue
                );

                cy.get(RiskPageSelectors.table.dataRows).should('have.length', staticCount);
            });
        });
    });
});
