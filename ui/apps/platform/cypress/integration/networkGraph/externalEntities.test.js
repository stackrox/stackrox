import { selectors as networkPageSelectors } from '../../constants/NetworkPage';
// TODO: import selectors from '../../selectors';

import withAuth from '../../helpers/basicAuth';
import {
    // TODO: clickOnNodeByName,
    // TODO: filterDeployments,
    filterNamespaces,
    filterClusters,
    filterInternet,
    visitNetworkGraphWithMockedData,
} from '../../helpers/networkGraph';

describe('External Entities on Network Graph', () => {
    withAuth();

    describe('Baseline state', () => {
        it('should group the namespaces into a cluster wrapper', () => {
            visitNetworkGraphWithMockedData();
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
            visitNetworkGraphWithMockedData();
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
