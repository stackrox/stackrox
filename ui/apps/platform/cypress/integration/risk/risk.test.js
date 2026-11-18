import withAuth from '../../helpers/basicAuth';
import navigationSelectors from '../../selectors/navigation';
import { hasFeatureFlag } from '../../helpers/features';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
import pf6 from '../../selectors/pf6';

import {
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

            const navText = hasFeatureFlag('ROX_UI_SECRETS_PAGE_MIGRATION') ? 'Workloads' : 'Risk';
            cy.get(`${pf6.navItem} a:contains("${navText}")`).should('have.class', 'pf-m-current');
        });

        it('should maintain active horizontal nav state on detail page for each view', () => {
            cy.wrap([
                { urlParam: 'Applications view', activeTab: 'User Workloads' },
                { urlParam: 'Platform view', activeTab: 'Platform' },
                { urlParam: 'Full view', activeTab: 'All Deployments' },
            ]).each(({ urlParam, activeTab }) => {
                visitRiskDeployments(urlParam);
                viewFirstRiskDeployment();

                cy.get(`${navigationSelectors.horizontalNavLinks}:contains("${activeTab}")`).should(
                    'have.class',
                    'pf-m-current'
                );
            });
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

            cy.get(`${pf6.tabButton}:contains("Risk indicators")`);
            cy.get(`${pf6.tabButton}:contains("Deployment details")`);
            cy.get(`${pf6.tabButton}:contains("Process discovery")`);
        });

        it('should navigate from Risk Page to Vulnerability Management Image Page', () => {
            visitRiskDeployments('Platform view');
            viewFirstRiskDeployment();

            cy.get(`${pf6.tabButton}:contains("Deployment details")`).click();

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

        it('should not have anything in search bar when URL has no search params', () => {
            visitRiskDeployments('Platform view');

            // Positive assertion: the compound search filter input is present.
            cy.get(RiskPageSelectors.search.input);
            // Negative assertion: no applied filter chips.
            cy.get(RiskPageSelectors.search.filterChipGroup).should('not.exist');
        });

        it('should render a chip for a single URL search param key/value pair', () => {
            const nsValue = 'stackrox';

            visitRiskDeploymentsWithSearchQuery('Platform view', `s[Namespace]=${nsValue}`);

            cy.get(RiskPageSelectors.search.filterChipGroup).should('have.length', 1);
            cy.get(RiskPageSelectors.search.filterChips).within(() => {
                // Chip group category is the attribute's filterChipLabel, plus the value chip.
                cy.contains('Namespace name');
                cy.contains('.pf-v6-c-label', nsValue);
            });
        });

        it('should render chips for multiple URL search param key/value pairs', () => {
            const nsValue = 'stackrox';
            const deployValue = 'scanner-v4-indexer';

            visitRiskDeploymentsWithSearchQuery(
                'Platform view',
                `s[Namespace]=${nsValue}&s[Deployment]=${deployValue}`
            );

            cy.get(RiskPageSelectors.search.filterChipGroup).should('have.length', 2);
            cy.get(RiskPageSelectors.search.filterChips).within(() => {
                cy.contains('Namespace name');
                cy.contains('.pf-v6-c-label', nsValue);
                cy.contains('Deployment name');
                cy.contains('.pf-v6-c-label', deployValue);
            });
        });
    });
});
