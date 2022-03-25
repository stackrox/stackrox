import { url as networkUrl, selectors as networkPageSelectors } from '../../constants/NetworkPage';

import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import {
    clickOnNodeByName,
    filterDeployments,
    filterNamespaces,
    selectNamespaceFilters,
} from '../../helpers/networkGraph';

describe('Network Deployment Details', () => {
    withAuth();

    beforeEach(() => {
        cy.server();

        cy.fixture('network/networkGraph.json').as('networkGraphJson');
        cy.route('GET', api.network.networkGraph, '@networkGraphJson').as('networkGraph');

        cy.fixture('network/networkPolicies.json').as('networkPoliciesJson');
        cy.route('GET', api.network.networkPoliciesGraph, '@networkPoliciesJson').as(
            'networkPolicies'
        );

        cy.fixture('network/centralDeployment.json').as('centralDeploymentJson');
        cy.route('GET', api.network.deployment, '@centralDeploymentJson').as('centralDeployment');

        cy.visit(networkUrl);

        selectNamespaceFilters('stackrox');

        cy.wait('@networkGraph');
        cy.wait('@networkPolicies');
    });

    it('should open up the Deployments Side Panel when a deployment is clicked', () => {
        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
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

    beforeEach(() => {
        cy.server();
        cy.route('GET', api.network.networkPoliciesGraph).as('networkPoliciesGraph');
        cy.route('GET', api.network.networkGraph).as('networkGraph');
    });

    it('should filter to show only the deployments from the stackrox namespace and deployments connected to them', () => {
        cy.visit(networkUrl);

        selectNamespaceFilters('stackrox');

        cy.wait(['@networkPoliciesGraph', '@networkGraph']);

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            const deployments = cytoscape.nodes().filter(filterDeployments);
            deployments.forEach((deployment) => {
                expect(deployment.data().parent).to.be.oneOf(['stackrox', 'kube-system']);
            });
        });
    });

    it('should filter to show only the stackrox namespace and deployments connected to stackrox namespace', () => {
        cy.visit(networkUrl);

        selectNamespaceFilters('stackrox', 'kube-system');

        cy.wait(['@networkPoliciesGraph', '@networkGraph']);

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            const namespaces = cytoscape.nodes().filter(filterNamespaces);
            expect(namespaces.size()).to.equal(1);
            namespaces.forEach((namespace) => {
                expect(namespace.data().name).to.be.oneOf(['stackrox', 'kube-system']);
            });
        });
    });

    it('should filter to show only a specific deployment and deployments connected to it', () => {
        const deploymentName = 'central';

        cy.visit(networkUrl);

        selectNamespaceFilters('stackrox', 'kube-system');

        cy.get(networkPageSelectors.toolbar.filterSelect).then(([$searchMultiSelect]) => {
            cy.wrap($searchMultiSelect).type('Deployment{enter}');
            cy.wrap($searchMultiSelect).type(`${deploymentName}{enter}`);
        });

        cy.wait(['@networkPoliciesGraph', '@networkGraph']);

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
