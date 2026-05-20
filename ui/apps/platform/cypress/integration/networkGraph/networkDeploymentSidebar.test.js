import withAuth from '../../helpers/basicAuth';

import {
    checkNetworkGraphEmptyState,
    selectCluster,
    selectDeployment,
    selectNamespace,
    visitNetworkGraph,
} from './networkGraph.helpers';
import { networkGraphSelectors } from './networkGraph.selectors';

describe('Network Graph deployment sidebar', () => {
    withAuth();

    it('should render a graph when cluster and namespace are selected', () => {
        visitNetworkGraph();

        checkNetworkGraphEmptyState();

        selectCluster();
        selectNamespace('stackrox');
        selectDeployment('sensor');

        // confirm that the graph shows sensor and its connections (central)
        cy.get(
            `${networkGraphSelectors.nodes} [data-type="node"] .pf-topology__node__label:contains("central")`,
            { timeout: 30000 }
        );

        // click on sensor node to open sidebar
        cy.get(`${networkGraphSelectors.nodes} [data-type="node"] .pf-topology__node__label`)
            .contains('sensor')
            .parent()
            .click();

        cy.get(networkGraphSelectors.drawerTitle).contains('sensor');
        cy.get(`${networkGraphSelectors.drawerSubtitle}:contains("stackrox")`);

        // check Details tab
        cy.get(`${networkGraphSelectors.drawerTabs} .pf-m-current`).contains('Details');

        cy.get('h2:contains("Network security")');
        cy.get('h2:contains("Deployment overview")');
        cy.get('h2:contains("Port configurations")');
        cy.get('h2:contains("Container configurations")');
    });
});
