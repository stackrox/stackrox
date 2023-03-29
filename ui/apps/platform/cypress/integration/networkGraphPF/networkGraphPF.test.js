import navigationSelectors from '../../selectors/navigation';

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

describe('Network page', () => {
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

    it('should render a graph when cluster and namespace are selected', () => {
        visitNetworkGraph();

        checkNetworkGraphEmptyState();

        selectCluster();
        selectNamespace('stackrox');

        // check that group of nodes for NS is present
        cy.get(
            '.pf-ri__topology-demo .pf-topology-content [data-id="stackrox-active-graph"] [data-layer-id="groups"] [data-id="stackrox"]'
        );

        // check that label for NS is present
        cy.get(
            '.pf-ri__topology-demo .pf-topology-content [data-id="stackrox-active-graph"] [data-layer-id="default"] [data-id="stackrox"] text'
        ).contains('stackro');
    });
});
