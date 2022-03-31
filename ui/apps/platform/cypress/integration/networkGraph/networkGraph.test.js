import * as api from '../../constants/apiEndpoints';
import { selectors as networkPageSelectors } from '../../constants/NetworkPage';
import withAuth from '../../helpers/basicAuth';
import {
    clickOnNodeByName,
    filterDeployments,
    filterNamespaces,
    selectDeploymentFilter,
    visitNetworkGraphWithMockedData,
    visitNetworkGraphWithNamespaceFilters,
} from '../../helpers/networkGraph';

describe('Network Deployment Details', () => {
    withAuth();

    it('should open up the Deployments Side Panel when a deployment is clicked', () => {
        visitNetworkGraphWithMockedData();

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            cy.intercept('GET', api.network.deployment, {
                fixture: 'network/centralDeployment.json',
            }).as('centralDeployment');
            clickOnNodeByName(cytoscape, {
                type: 'DEPLOYMENT',
                name: 'central',
            });
            cy.wait('@centralDeployment');
            cy.get(`${networkPageSelectors.networkEntityTabbedOverlay.header}:contains("central")`);
        });
    });
});

describe('Network Graph Search', () => {
    withAuth();

    it('should filter to show only the deployments from the stackrox namespace and deployments connected to them', () => {
        visitNetworkGraphWithNamespaceFilters('stackrox');

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            const deployments = cytoscape.nodes().filter(filterDeployments);
            deployments.forEach((deployment) => {
                expect(deployment.data().parent).to.be.oneOf(['stackrox', 'kube-system']);
            });
        });
    });

    it('should filter to show only the stackrox namespace and deployments connected to stackrox namespace', () => {
        visitNetworkGraphWithNamespaceFilters('stackrox', 'kube-system');

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            const namespaces = cytoscape.nodes().filter(filterNamespaces);
            expect(namespaces.size()).to.equal(1);
            namespaces.forEach((namespace) => {
                expect(namespace.data().name).to.be.oneOf(['stackrox', 'kube-system']);
            });
        });
    });

    it('should filter to show only a specific deployment and deployments connected to it', () => {
        visitNetworkGraphWithNamespaceFilters('stackrox', 'kube-system');

        selectDeploymentFilter('central');

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            const deployments = cytoscape.nodes().filter(filterDeployments);
            expect(deployments.size()).to.be.at.least(3); // central, scanner, sensor

            const minDeps = [];
            deployments.forEach((deployment) => {
                minDeps.push(deployment.data().name);
            });
            expect(minDeps).to.include.members(['central', 'scanner', 'sensor']);
        });
    });
});
