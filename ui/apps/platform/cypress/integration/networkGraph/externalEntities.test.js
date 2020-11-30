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

    beforeEach(() => {
        cy.server();

        cy.fixture('network/networkGraph.json').as('networkGraphJson');
        cy.route('GET', api.network.networkGraph, '@networkGraphJson').as('networkGraph');

        cy.fixture('network/networkPolicies.json').as('networkPoliciesJson');
        cy.route('GET', api.network.networkPoliciesGraph, '@networkPoliciesJson').as(
            'networkPolicies'
        );

        cy.visit(networkUrl);
        cy.wait('@networkGraph');
        cy.wait('@networkPolicies');
    });

    describe('Baseline state', () => {
        it('should group the namespaces into a cluster wrapper', () => {
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
