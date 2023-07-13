import withAuth from '../../helpers/basicAuth';
import {
    assertSortedItems,
    callbackForPairOfAscendingNumberValuesFromElements,
    callbackForPairOfDescendingNumberValuesFromElements,
} from '../../helpers/sort';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    clickTab,
    deploymentswithprocessinfoAlias,
    deploymentscountAlias,
    visitRiskDeployments,
    visitRiskDeploymentsWithSearchQuery,
    viewRiskDeploymentByName,
    viewRiskDeploymentInNetworkGraph,
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

            cy.get('.rt-th:contains("Name")');
            cy.get('.rt-th:contains("Created")');
            cy.get('.rt-th:contains("Cluster")');
            cy.get('.rt-th:contains("Namespace")');
            cy.get('.rt-th:contains("Priority")');
        });

        /*
         * ROX-13468: assertSortedItems sometimes fails for sort descending (step 2) or resort ascending (step 3).
         * This is the only sort test that fails and Risk is the only occurrence of TableV2 element.
         * Skip test given the comment below (step 0) initial table state and other possible rendering problems.
         */
        it.skip('should sort the Priority column', () => {
            visitRiskDeployments();

            const thSelector = '.rt-th:contains("Priority")';
            const tdSelector = '.rt-td:nth-child(5)';

            // 0. Initial table state does not indicate that it is sorted ascending by the Priority column.
            // TODO If possible, replace TableV2 with Table element in RiskTable component.
            cy.get(thSelector)
                .should('not.have.class', '-sort-asc')
                .should('not.have.class', '-sort-desc');

            // 1. Sort ascending by the Priority column.
            cy.get(thSelector).click();
            cy.location('search').should(
                'eq',
                '?sort[id]=Deployment%20Risk%20Priority&sort[desc]=false'
            );

            // There is no request because rows are already sorted ascending.

            cy.get(thSelector).should('have.class', '-sort-asc');
            cy.get(tdSelector).then((items) => {
                assertSortedItems(items, callbackForPairOfAscendingNumberValuesFromElements);
            });

            // 2. Sort descending by the Priority column.
            cy.get(thSelector).click();
            cy.location('search').should(
                'eq',
                '?sort[id]=Deployment%20Risk%20Priority&sort[desc]=true'
            );

            // There is a request because of change in sorting.
            cy.wait(`@${deploymentswithprocessinfoAlias}`)
                .its('request.url')
                .should('include', 'sortOption.field=Deployment%20Risk%20Priority')
                .should('include', 'sortOption.reversed=true');
            cy.wait(`@${deploymentscountAlias}`);

            cy.get(thSelector).should('have.class', '-sort-desc');
            cy.get(tdSelector).then((items) => {
                assertSortedItems(items, callbackForPairOfDescendingNumberValuesFromElements);
            });

            // 3. Sort ascending by the Priority column.
            cy.get(thSelector).click();
            cy.location('search').should(
                'eq',
                '?sort[id]=Deployment%20Risk%20Priority&sort[desc]=false'
            );

            // There is a request because of change in sorting.
            cy.wait(`@${deploymentswithprocessinfoAlias}`)
                .its('request.url')
                .should('include', 'sortOption.field=Deployment%20Risk%20Priority')
                .should('include', 'sortOption.reversed=false');
            cy.wait(`@${deploymentscountAlias}`);

            cy.get(thSelector).should('have.class', '-sort-asc');
            cy.get(tdSelector).then((items) => {
                assertSortedItems(items, callbackForPairOfAscendingNumberValuesFromElements);
            });
        });

        it('should open side panel for deployment', () => {
            visitRiskDeployments();
            viewRiskDeploymentByName('collector');
        });

        // TODO add relevant tests for error messages in PatternFly

        it('should open the panel to view risk indicators', () => {
            visitRiskDeployments();
            viewRiskDeploymentByName('collector');

            cy.get(RiskPageSelectors.panel).should('have.length', 2); // main panel and side panel
            cy.get('button[data-testid="tab"]:contains("Risk Indicators")');
            cy.get('button[aria-label="Close"]').click();
            cy.get(RiskPageSelectors.panel).should('have.length', 1); // main panel
        });

        it('should open the panel to view deployment details', () => {
            visitRiskDeployments();
            viewRiskDeploymentByName('collector');

            cy.get(RiskPageSelectors.panel).should('have.length', 2); // main panel and side panel
            cy.get('button[data-testid="tab"]:contains("Deployment Details")');
            cy.get('button[aria-label="Close"]').click();
            cy.get(RiskPageSelectors.panel).should('have.length', 1); // main panel
        });

        it('should navigate from Risk Page to Vulnerability Management Image Page', () => {
            visitRiskDeployments();
            viewRiskDeploymentByName('collector');

            clickTab('Deployment Details');
            cy.get(RiskPageSelectors.imageLink).first().click();
            cy.location('pathname').should('contain', '/main/vulnerability-management/image');
        });
    });

    describe('with actual API', () => {
        // TODO fix uncaught exception in Network Graph 2.0
        it.skip('should navigate to network page with selected deployment', () => {
            visitRiskDeployments();
            viewRiskDeploymentByName('collector');
            viewRiskDeploymentInNetworkGraph();
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
                `${RiskPageSelectors.table.dataRows} .rt-td:nth-child(4):contains("${nsValue}")`
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
                `${RiskPageSelectors.table.dataRows} .rt-td:nth-child(1):contains("${deployValue}")`
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

        it('should not use invalid URL search param key/value pair in its search bar', () => {
            visitRiskDeployments();

            const sillyOption = 'Wingardium';
            const sillyValue = 'leviosa';
            cy.get(RiskPageSelectors.table.dataRows).then((allDeps) => {
                const allCount = allDeps.length;

                visitRiskDeploymentsWithSearchQuery(`?s[${sillyOption}]=${sillyValue}`);

                // Positive assertion:
                cy.get(searchPlaceholderSelector);
                // Negative assertion:
                cy.get(RiskPageSelectors.search.searchLabels).should('not.exist');

                cy.get(RiskPageSelectors.table.dataRows).should('have.length', allCount);
            });
        });
    });
});
