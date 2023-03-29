import navigationSelectors from '../../selectors/navigation';
import { networkGraphSelectors } from './networkGraph.selectors';

import withAuth from '../../helpers/basicAuth';
import {
    visitNetworkGraph,
    visitNetworkGraphFromLeftNav,
    checkNetworkGraphEmptyState,
    selectCluster,
    selectNamespace,
} from '../../helpers/networkGraphPF';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
import { hasFeatureFlag } from '../../helpers/features';

describe('Network Graph smoke tests', () => {
    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_NETWORK_GRAPH_PATTERNFLY')) {
            this.skip();
        }
    });

    withAuth();

    it('should visit using the left nav', () => {
        visitNetworkGraphFromLeftNav();

        cy.get(`${navigationSelectors.navLinks}:contains('Network Graph')`)
            .first()
            .should('have.class', 'pf-m-current');

        checkNetworkGraphEmptyState();
    });

    it('should visit from direct navigation', () => {
        visitNetworkGraph();

        cy.title().should('match', getRegExpForTitleWithBranding('Network Graph'));

        checkNetworkGraphEmptyState();
    });

    it('should render a graph, including toolbar, when cluster and namespace are selected', () => {
        visitNetworkGraph();

        checkNetworkGraphEmptyState();

        selectCluster();
        selectNamespace('stackrox');

        // check that group of nodes for NS is present
        cy.get(`${networkGraphSelectors.groups} [data-id="stackrox"]`);

        // check that label for NS is present and has the filtered-namespace class
        cy.get(
            `${networkGraphSelectors.nodes} [data-id="stackrox"] g.filtered-namespace text`
        ).contains('stackrox');

        // check that toolbar and buttons are present
        cy.get(`${networkGraphSelectors.toolbar}`);
        cy.get(networkGraphSelectors.toolbarItem).contains('Zoom In');
        cy.get(networkGraphSelectors.toolbarItem).contains('Zoom Out');
        cy.get(networkGraphSelectors.toolbarItem).contains('Fit to Screen');
        cy.get(networkGraphSelectors.toolbarItem).contains('Reset View');

        // open Legend as well, after verifying its existence
        cy.get(networkGraphSelectors.toolbarItem).contains('Legend').click();

        // check Legend content
        cy.get('.pf-c-popover__content [data-testid="legend-title"]:contains("Legend")');

        cy.get('.pf-c-popover__content [data-testid="node-types-title"]:contains("Node types")');
        cy.get('.pf-c-popover__content .pf-c-description-list__text:contains("Deployment")');
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("External CIDR block")'
        );

        cy.get(
            '.pf-c-popover__content [data-testid="namespace-types-title"]:contains("Namespace types")'
        );
        cy.get('.pf-c-popover__content .pf-c-description-list__text:contains("Derived namespace")');
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("Filtered namespace")'
        );

        cy.get(
            '.pf-c-popover__content [data-testid="deployment-badges-title"]:contains("Deployment badges")'
        );
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("Connected to external entities")'
        );
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("Isolated by network policy rules")'
        );
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("All traffic allowed (No network policies)")'
        );
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("Only has an egress network policy")'
        );
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("Only has an ingress network policy")'
        );

        // close the Legend
        cy.get('.pf-c-popover__content [aria-label="Close"]').click();
    });
});
