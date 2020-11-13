import { url as networkUrl, selectors as networkPageSelectors } from '../../constants/NetworkPage';
// TODO: import selectors from '../../selectors';

import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import checkFeatureFlag from '../../helpers/features';
import {
    // TODO:    clickOnNodeByName,
    // TODO: filterDeployments,
    filterNamespaces,
    filterClusters,
    filterInternet,
} from '../../helpers/networkGraph';

describe('External Entities on Network Graph', () => {
    before(function beforeHook() {
        if (checkFeatureFlag('ROX_NETWORK_GRAPH_EXTERNAL_SRCS', false)) {
            this.skip();
        }
    });

    withAuth();

    describe('Baseline state', () => {
        beforeEach(() => {
            cy.server();
            cy.route('GET', api.network.networkPoliciesGraph).as('networkPoliciesGraph');
        });

        it('should group the namespaces into a cluster wrapper', () => {
            cy.visit(networkUrl);
            cy.wait('@networkPoliciesGraph');
            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const clusters = cytoscape.nodes().filter(filterClusters);
                expect(clusters.size()).to.equal(1);
                const clusterData = clusters[0].data();
                expect(clusterData.name).to.eq('remote');

                const clusterChildren = clusters[0].children();
                const namespaces = cytoscape.nodes().filter(filterNamespaces);
                expect(clusterChildren.contains(namespaces)).to.be.true;
            });
        });

        it('should group external connections into a node outside the cluster', () => {
            cy.visit(networkUrl);
            cy.wait('@networkPoliciesGraph');
            cy.wait(2000); // extending the timeout on the following get is not enough
            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const externalEntities = cytoscape.nodes().filter(filterInternet);
                expect(externalEntities.size()).to.equal(1);
                const externalEntitiesData = externalEntities[0].data();
                expect(externalEntitiesData.name).to.eq('External Entities');

                const clusters = cytoscape.nodes().filter(filterClusters);
                const clusterChildren = clusters[0].children();
                expect(clusterChildren.contains(externalEntities)).not.to.be.true;
            });
        });
    });
});
