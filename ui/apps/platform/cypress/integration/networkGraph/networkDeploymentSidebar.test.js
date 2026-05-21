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

        // Verify sensor node exists in the namespace graph (no deployment filter)
        cy.get(networkGraphSelectors.deploymentNode('sensor'), { timeout: 30000 });

        // Click on sensor node to open sidebar (force: nodes may overlap in topology)
        cy.get(networkGraphSelectors.deploymentNode('sensor')).click({ force: true });

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
