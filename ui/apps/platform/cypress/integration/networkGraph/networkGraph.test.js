import { url as networkUrl, selectors as networkPageSelectors } from '../../constants/NetworkPage';
import selectors from '../../selectors';

import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { clickOnNodeByName, filterDeployments, filterNamespaces } from '../../helpers/networkGraph';

describe('Network Deployment Details', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route('GET', api.network.networkPoliciesGraph).as('networkPoliciesGraph');
    });

    it('should open up the Deployments Side Panel when a deployment is clicked', () => {
        cy.visit(networkUrl);
        cy.wait('@networkPoliciesGraph');
        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            clickOnNodeByName(cytoscape, {
                type: 'DEPLOYMENT',
                name: 'central',
            });
            cy.get(`${networkPageSelectors.detailsPanel.header}:contains("central")`);
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
