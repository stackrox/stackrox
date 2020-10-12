import { url as networkUrl, selectors as networkPageSelectors } from '../constants/NetworkPage';
import { url as riskURL, selectors as riskPageSelectors } from '../constants/RiskPage';

import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';
import selectors from '../selectors/index';
import checkFeatureFlag from '../helpers/features';

function uploadYAMLFile(fileName, selector) {
    cy.fixture(fileName).then((fileContent) => {
        cy.get(selector).upload({ fileContent, fileName, mimeType: 'text/yaml', encoding: 'utf8' });
    });
}

function navigateToNetworkGraphWithMockedData() {
    cy.server();
    cy.fixture('network/networkGraph.json').as('networkGraphJson');
    cy.route('GET', api.network.networkPoliciesGraph, '@networkGraphJson').as('networkGraph');
    cy.visit(networkUrl);
    cy.wait('@networkGraph');
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

        cy.get(networkPageSelectors.panels.detailsPanel, { timeout: 15000 }).should('be.visible');
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

    it('should let you navigate to a different deployment', () => {
        cy.visit(riskURL);
        cy.get(`${selectors.table.rows}:contains('central')`).click();
        cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
        cy.get(networkPageSelectors.networkDetailsPanel.header).contains('central');
        cy.wait(1500);
        cy.get(`${selectors.table.rows}:eq(0) .hidden button`)
            .invoke('show')
            .click({ force: true });
        cy.get(networkPageSelectors.networkDetailsPanel.header).should('not.contain', 'central');
    });

    it('should show the proper table column headers for the network flows table', () => {
        cy.visit(riskURL);
        cy.get(`${selectors.table.rows}:contains('central')`).click();
        cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
        cy.get(`${selectors.tab.tabs}:contains('Network Flows')`).click();
        cy.get(`${selectors.table.columnHeaders}:contains('Traffic')`);
        cy.get(`${selectors.table.columnHeaders}:contains('Deployment')`);
        cy.get(`${selectors.table.columnHeaders}:contains('Namespace')`);
        cy.get(`${selectors.table.columnHeaders}:contains('Connection')`);

        if (checkFeatureFlag('ROX_NETWORK_GRAPH_PORTS', true)) {
            cy.get(`${selectors.table.columnHeaders}:contains('Protocols')`);
            cy.get(`${selectors.table.columnHeaders}:contains('Ports')`);
        }
    });

    // eslint-disable-next-line func-names
    it('should show the proper client-side autocomplete results for network flows table', function () {
        if (checkFeatureFlag('ROX_NETWORK_FLOWS_SEARCH_FILTER_UI', false)) {
            this.skip();
        }

        cy.visit(riskURL);
        cy.get(`${selectors.table.rows}:contains('central')`).click();
        cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
        cy.get(`${selectors.tab.tabs}:contains('Network Flows')`).click();

        // check autocomplete results for Traffic
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Traffic:{enter}');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('ingress');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('bidirectional');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('egress');

        // check autocomplete results for Deployment names
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Deployment:{enter}');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('sensor');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('scanner');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('monitoring');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('kube-dns');

        // check autocomplete results for Namespace names
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Namespace:{enter}');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('stackrox');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('kube-system');

        // check autocomplete results for Protocols
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Protocols:{enter}');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('L4_PROTOCOL_TCP');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('L4_PROTOCOL_UDP');

        // check autocomplete results for Ports
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Ports:{enter}');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('8443');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('8080');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('53');

        // check autocomplete results for Connections
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Connection:{enter}');
        cy.get(networkPageSelectors.networkFlowsSearch.options).contains('active');
    });

    // eslint-disable-next-line func-names
    it('should properly filter the network flows table', function () {
        if (checkFeatureFlag('ROX_NETWORK_FLOWS_SEARCH_FILTER_UI', false)) {
            this.skip();
        }

        cy.visit(riskURL);
        cy.get(`${selectors.table.rows}:contains('central')`).click();
        cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
        cy.get(`${selectors.tab.tabs}:contains('Network Flows')`).click();

        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Traffic:{enter}');
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('ingress{enter}');
        cy.get(selectors.table.rows).each(($el) => {
            cy.wrap($el).get(`${selectors.table.cells}:eq(1)`).contains('ingress');
        });

        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Deployment:{enter}');
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('sensor{enter}');
        cy.get(selectors.table.rows).each(($el) => {
            cy.wrap($el).get(`${selectors.table.cells}:eq(2)`).contains('sensor');
        });

        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Namespace:{enter}');
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('kube-system{enter}');
        cy.get(selectors.table.rows).each(($el) => {
            cy.wrap($el).get(`${selectors.table.cells}:eq(3)`).contains('kube-system');
        });

        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Protocols:{enter}');
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('L4_PROTOCOL_TCP{enter}');
        cy.get(selectors.table.rows).each(($el) => {
            cy.wrap($el).get(`${selectors.table.cells}:eq(4)`).contains('TCP');
        });

        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Ports:{enter}');
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('8443{enter}');
        cy.get(selectors.table.rows).each(($el) => {
            cy.wrap($el)
                .get(`${selectors.table.cells}:eq(5)`)
                .contains(/8443|Multiple/g);
        });

        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Connection:{enter}');
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('active{enter}');
        cy.get(selectors.table.rows).each(($el) => {
            cy.wrap($el).get(`${selectors.table.cells}:eq(6)`).contains('active');
        });
    });
});
