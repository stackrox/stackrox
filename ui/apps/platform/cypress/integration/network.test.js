import { url as networkUrl, selectors as networkPageSelectors } from '../constants/NetworkPage';
import { url as riskURL, selectors as riskPageSelectors } from '../constants/RiskPage';

import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';
import selectors from '../selectors/index';
import checkFeatureFlag from '../helpers/features';

function uploadYAMLFile(fileName, selector) {
    cy.fixture(fileName).then((fileContent) => {
        cy.get(selector).attachFile({
            fileContent,
            fileName,
            mimeType: 'text/yaml',
            encoding: 'utf8',
        });
    });
}

function navigateToNetworkGraphWithMockedData() {
    cy.server();

    cy.fixture('network/networkGraph.json').as('networkGraphJson');
    cy.route('GET', api.network.networkGraph, '@networkGraphJson').as('networkGraph');

    cy.fixture('network/networkPolicies.json').as('networkPoliciesJson');
    cy.route('GET', api.network.networkPoliciesGraph, '@networkPoliciesJson').as('networkPolicies');

    cy.visit(networkUrl);
    cy.wait('@networkGraph');
    cy.wait('@networkPolicies');
}

describe('Network page', () => {
    withAuth();

    it('should have selected item in nav bar', () => {
        navigateToNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.network).click();
        cy.get(networkPageSelectors.network).should('have.class', 'bg-primary-700');
    });

    it('should display a legend', () => {
        navigateToNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.legend.deployments)
            .eq(0)
            .children()
            .should('have.class', 'icon-node');

        if (checkFeatureFlag('ROX_NETWORK_GRAPH_EXTERNAL_SRCS', true)) {
            cy.get(networkPageSelectors.legend.deployments)
                .eq(1)
                .children()
                .should('have.attr', 'alt', 'deployment-external-connections');
            cy.get(networkPageSelectors.legend.deployments)
                .eq(2)
                .children()
                .children()
                .should('have.class', 'icon-potential');
            cy.get(networkPageSelectors.legend.deployments)
                .eq(3)
                .children()
                .should('have.class', 'icon-node');
        } else {
            cy.get(networkPageSelectors.legend.deployments)
                .eq(1)
                .children()
                .children()
                .should('have.class', 'icon-potential');
            cy.get(networkPageSelectors.legend.deployments)
                .eq(2)
                .children()
                .should('have.class', 'icon-node');
        }

        cy.get(networkPageSelectors.legend.namespaces)
            .eq(0)
            .children()
            .should('have.attr', 'alt', 'namespace');
        cy.get(networkPageSelectors.legend.namespaces)
            .eq(1)
            .children()
            .should('have.attr', 'alt', 'namespace-allowed-connection');
        cy.get(networkPageSelectors.legend.namespaces)
            .eq(2)
            .children()
            .should('have.attr', 'alt', 'namespace-connection');

        cy.get(networkPageSelectors.legend.connections)
            .eq(0)
            .children()
            .should('have.attr', 'alt', 'active-connection');
        cy.get(networkPageSelectors.legend.connections)
            .eq(1)
            .children()
            .should('have.attr', 'alt', 'allowed-connection');
        cy.get(networkPageSelectors.legend.connections)
            .eq(2)
            .children()
            .should('have.class', 'icon-ingress-egress');
    });

    it('should handle toggle click on simulator network policy button', () => {
        navigateToNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        cy.get(networkPageSelectors.buttons.viewActiveYamlButton).should('be.visible');
        cy.get(networkPageSelectors.panels.creatorPanel).should('be.visible');
        cy.get(networkPageSelectors.buttons.simulatorButtonOn).click();
        cy.get(networkPageSelectors.panels.creatorPanel).should('not.be.visible');
    });

    it('should display error messages when uploaded wrong yaml', () => {
        navigateToNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        uploadYAMLFile('network/policywithoutnamespace.yaml', 'input[type="file"]');

        cy.get(networkPageSelectors.simulatorSuccessMessage).should('not.be.visible');
    });

    it('should display success messages when uploaded right yaml', () => {
        navigateToNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        uploadYAMLFile('network/policywithnamespace.yaml', 'input[type="file"]');

        cy.get(networkPageSelectors.simulatorSuccessMessage).should('be.visible');
    });

    it('should show the network policy simulator screen after generating network policies', () => {
        cy.visit(riskURL);
        cy.get(selectors.table.rows).eq(0).click({ force: true });
        cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton, { timeout: 10000 }).click();

        cy.get(networkPageSelectors.networkEntityTabbedOverlay.header, { timeout: 15000 }).should(
            'be.visible'
        );
        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        cy.get(networkPageSelectors.buttons.generateNetworkPolicies).click();
        cy.get(networkPageSelectors.panels.simulatorPanel, { timeout: 10000 }).should('be.visible');
    });
});

describe('Network Deployment Details', () => {
    withAuth();

    it('should show the port exposure levels using port configuration labels', () => {
        cy.visit(riskURL);
        cy.get(`${selectors.table.rows}:contains('central')`).click();
        cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
        cy.get(`${selectors.tab.tabs}:contains('Details')`).click();
        cy.get(`[data-testid="exposure"]:contains('ClusterIP')`);
        cy.get(`[data-testid="level"]:contains('ClusterIP')`);
    });
});

describe('Network Policy Simulator', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route('POST', api.network.simulate).as('simulateGraph');
    });

    it('should update the graph when generating and simulating network policies', () => {
        // this will get the deployments for the 'default' and 'docker' namespace
        function getDeployments(cytoscape) {
            const deployments = cytoscape.filter((element) => {
                return (
                    element.isNode() &&
                    element.data('type') === 'DEPLOYMENT' &&
                    (element.data('parent') === 'default' || element.data('parent') === 'docker')
                );
            });
            return deployments;
        }

        cy.visit(networkUrl);
        cy.get(networkPageSelectors.buttons.allowedFilter).click();
        cy.getCytoscape('#cytoscapeContainer').then((cytoscape) => {
            const deployments = getDeployments(cytoscape);
            // we want to make sure all the deployments from 'default' and 'docker' namespaces are non-isolated
            deployments.forEach((deployment) => {
                expect(deployment.hasClass('nonIsolated')).to.equal(true);
            });
            cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
            cy.get(networkPageSelectors.buttons.generateNetworkPolicies).click();
            // wait for the graph to update with the new data
            cy.wait('@simulateGraph');
            cy.getCytoscape('#cytoscapeContainer').then((updatedCytoscape) => {
                const simulatedDeployments = getDeployments(updatedCytoscape);
                // After the simulated graph, we want to make sure all the deployments from 'default' and 'docker' namespaces are not non-isolated
                simulatedDeployments.forEach((deployment) => {
                    expect(deployment.hasClass('nonIsolated')).to.equal(false);
                });
            });
        });
    });
});

describe('Network Flows Table', () => {
    withAuth();

    it('should show the proper table column headers for the network flows table', () => {
        cy.visit(riskURL);
        cy.get(`${selectors.table.rows}:contains('central')`).click();
        cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
        cy.get(`${selectors.tab.tabs}:contains('Network Flows')`).click();
        cy.get(`${selectors.table.th}:contains('Entity')`);
        cy.get(`${selectors.table.th}:contains('Traffic')`);
        cy.get(`${selectors.table.th}:contains('Type')`);
        cy.get(`${selectors.table.th}:contains('Namespace')`);
        cy.get(`${selectors.table.th}:contains('State')`);

        if (checkFeatureFlag('ROX_NETWORK_GRAPH_PORTS', true)) {
            cy.get(`${selectors.table.th}:contains('Protocol')`);
            cy.get(`${selectors.table.th}:contains('Port')`);
        }
    });
});
