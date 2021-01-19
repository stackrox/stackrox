import { url as networkUrl, selectors as networkPageSelectors } from '../../constants/NetworkPage';
import selectors from '../../selectors';

import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { clickOnNodeByName, filterDeployments, filterNamespaces } from '../../helpers/networkGraph';

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

    it('should filter to show only the deployments from the stackrox namespace', () => {
        const namespaceName = 'stackrox';

        cy.visit(networkUrl);
        cy.wait(['@networkPoliciesGraph', '@networkGraph']);

        cy.get(selectors.search.multiSelectInput).type('Namespace{enter}');
        cy.get(selectors.search.multiSelectInput).type(`${namespaceName}{enter}`);
        cy.wait(['@networkPoliciesGraph', '@networkGraph']);

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            const deployments = cytoscape.nodes().filter(filterDeployments);
            deployments.forEach((deployment) => {
                expect(deployment.data().parent).to.equal('stackrox');
            });
        });
    });

    it('should filter to show only the stackrox namespace', () => {
        const namespaceName = 'stackrox';

        cy.visit(networkUrl);
        cy.wait(['@networkPoliciesGraph', '@networkGraph']);

        cy.get(selectors.search.multiSelectInput).type('Namespace{enter}');
        cy.get(selectors.search.multiSelectInput).type(`${namespaceName}{enter}`);
        cy.wait(['@networkPoliciesGraph', '@networkGraph']);

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            const namespaces = cytoscape.nodes().filter(filterNamespaces);
            expect(namespaces.size()).to.equal(1);
            namespaces.forEach((namespace) => {
                expect(namespace.data().name).to.equal('stackrox');
            });
        });
    });

    it('should filter to show only a specific deployment', () => {
        const deploymentName = 'central';

        cy.visit(networkUrl);
        cy.wait(['@networkPoliciesGraph', '@networkGraph']);

        cy.get(selectors.search.multiSelectInput).type('Deployment{enter}');
        cy.get(selectors.search.multiSelectInput).type(`${deploymentName}{enter}`);
        cy.wait(['@networkPoliciesGraph', '@networkGraph']);

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            const deployments = cytoscape.nodes().filter(filterDeployments);
            expect(deployments.size()).to.equal(1);
            deployments.forEach((deployment) => {
                expect(deployment.data().name).to.equal('central');
            });
        });
    });
});
