import * as api from '../../constants/apiEndpoints';
import { selectors as networkPageSelectors } from '../../constants/NetworkPage';
import withAuth from '../../helpers/basicAuth';
import {
    clickOnNodeByName,
    filterDeployments,
    filterNamespaces,
    selectDeploymentFilter,
    selectNamespaceFilter,
    selectNamespaceFilterWithNetworkGraphResponse,
    visitOldNetworkGraph,
    visitOldNetworkGraphWithNamespaceFilter,
} from '../../helpers/networkGraph';

describe('Network Deployment Details', () => {
    withAuth();

    it('should open up the Deployments Side Panel when a deployment is clicked', () => {
        visitOldNetworkGraphWithNamespaceFilter('stackrox');

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            clickOnNodeByName(cytoscape, {
                type: 'DEPLOYMENT',
                name: 'central',
            });
            cy.get(`${networkPageSelectors.networkEntityTabbedOverlay.header}:contains("central")`);
        });
    });
});

describe('Network Graph Search', () => {
    withAuth();

    it('should filter to show only the deployments from the stackrox namespace and deployments connected to them', () => {
        visitOldNetworkGraphWithNamespaceFilter('stackrox');

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            const deployments = cytoscape.nodes().filter(filterDeployments);
            deployments.forEach((deployment) => {
                expect(deployment.data().parent).to.be.oneOf(['stackrox']);
            });
        });
    });

    it('should filter to show only the stackrox namespace and deployments connected to stackrox namespace', () => {
        visitOldNetworkGraphWithNamespaceFilter('stackrox');

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            const namespaces = cytoscape.nodes().filter(filterNamespaces);
            // For now, let the assertion pass even if array is empty.
            namespaces.forEach((namespace) => {
                expect(namespace.data().name).to.be.oneOf(['stackrox']);
            });
        });
    });

    it('should filter to show only a specific deployment and deployments connected to it', () => {
        visitOldNetworkGraphWithNamespaceFilter('stackrox');
        selectDeploymentFilter('sensor');

        const deploymentsExpected = ['admission-control', 'central', 'collector'];

        cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
            const deploymentsReceived = cytoscape
                .nodes()
                .filter(filterDeployments)
                .map((deployment) => deployment.data().name);
            expect(deploymentsReceived).to.include.members(deploymentsExpected);
        });
    });

    it('should render an error message when the server fails to return a successful response', () => {
        visitOldNetworkGraph();

        // Stub out an error response from the server
        const error =
            'Number of deployments (2200) exceeds maximum allowed for Network Graph: 2000';
        const response = {
            statusCode: 500,
            body: { error, message: error },
        };
        selectNamespaceFilterWithNetworkGraphResponse('stackrox', response);

        cy.get(networkPageSelectors.errorOverlay.heading);
        cy.get(networkPageSelectors.errorOverlay.message(error));

        // Ignore previously stubbed error response and allow the request to respond normally
        cy.intercept('GET', api.network.networkGraph, (req) => req.continue()).as('networkGraph');
        selectNamespaceFilter('kube-system');

        cy.get(networkPageSelectors.errorOverlay.heading).should('not.exist');
        cy.get(networkPageSelectors.cytoscapeContainer);
    });
});
