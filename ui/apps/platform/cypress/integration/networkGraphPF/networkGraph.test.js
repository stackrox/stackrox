import navigationSelectors from '../../selectors/navigation';

import withAuth from '../../helpers/basicAuth';
import { visitNetworkGraph, visitNetworkGraphFromLeftNav } from '../../helpers/networkGraphPF';
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
    });

    it('should have selected item in nav bar', () => {
        visitNetworkGraph();
        cy.get(`${navigationSelectors.navLinks}:contains('PatternFly Network Graph')`).should(
            'have.class',
            'pf-m-current'
        );
    });

    it('should have title', () => {
        visitNetworkGraph();

        cy.title().should('match', getRegExpForTitleWithBranding('Network Graph'));
    });
});
