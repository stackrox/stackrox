import { url as networkUrl, selectors as networkPageSelectors } from '../constants/NetworkPage';
import { url as riskURL, selectors as RiskPageSelectors } from '../constants/RiskPage';
import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';
import selectors from '../selectors/index';

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
        cy.get(RiskPageSelectors.networkNodeLink, { timeout: 10000 }).click();

        cy.get(networkPageSelectors.panels.detailsPanel).should('be.visible');
        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        cy.get(networkPageSelectors.buttons.generateNetworkPolicies).click();
        cy.get(networkPageSelectors.panels.simulatorPanel, { timeout: 10000 }).should('be.visible');
    });
});
