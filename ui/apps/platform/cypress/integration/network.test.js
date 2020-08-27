import { url as networkUrl, selectors as networkPageSelectors } from '../constants/NetworkPage';
import { url as riskURL, selectors as RiskPageSelectors } from '../constants/RiskPage';
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
    cy.route('GET', api.network.networkGraph, '@networkGraphJson').as('networkGraph');
    cy.visit(networkUrl);
    cy.wait('@networkGraph');
}

xdescribe('Network page', () => {
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
        cy.get(networkPageSelectors.legend.deployments)
            .eq(1)
            .children()
            .children()
            .should('have.class', 'icon-potential');
        cy.get(networkPageSelectors.legend.deployments)
            .eq(2)
            .children()
            .should('have.class', 'icon-node');

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
        cy.get(RiskPageSelectors.viewDeploymentsInNetworkGraphButton, { timeout: 10000 }).click();

        cy.get(networkPageSelectors.panels.detailsPanel).should('be.visible');
        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        cy.get(networkPageSelectors.buttons.generateNetworkPolicies).click();
        cy.get(networkPageSelectors.panels.simulatorPanel, { timeout: 10000 }).should('be.visible');
    });
});

describe('Network Flows Table', () => {
    withAuth();

    it('should show the proper table column headers for the network flows table', () => {
        cy.visit(riskURL);
        cy.get(`${selectors.table.rows}:contains('central')`).click();
        cy.get(RiskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
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
        cy.get(RiskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
        cy.get(`${selectors.tab.tabs}:contains('Network Flows')`).click();

        // check autocomplete results for Traffic
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Traffic:{enter}');
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(0)`).contains('ingress');
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(1)`).contains(
            'bidirectional'
        );
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(2)`).contains('egress');

        // check autocomplete results for Deployment names
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Deployment:{enter}');
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(0)`).contains('sensor');
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(1)`).contains('scanner');
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(2)`).contains('coredns');

        // check autocomplete results for Namespace names
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Namespace:{enter}');
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(0)`).contains('stackrox');
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(1)`).contains('kube-system');

        // check autocomplete results for Protocols
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Protocols:{enter}');
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(0)`).contains(
            'L4_PROTOCOL_TCP'
        );
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(1)`).contains(
            'L4_PROTOCOL_UDP'
        );

        // check autocomplete results for Ports
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Ports:{enter}');
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(0)`).contains('8443');
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(1)`).contains('8080');
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(2)`).contains('53');

        // check autocomplete results for Connections
        cy.get(networkPageSelectors.networkFlowsSearch.input).clear();
        cy.get(networkPageSelectors.networkFlowsSearch.input).type('Connection:{enter}');
        cy.get(`${networkPageSelectors.networkFlowsSearch.options}:eq(0)`).contains('active');
    });
});
