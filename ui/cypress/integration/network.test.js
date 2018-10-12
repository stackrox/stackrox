import { url as networkUrl, selectors as networkPageSelectors } from './constants/NetworkPage';
import * as api from './constants/apiEndpoints';

const uploadFile = (fileName, selector) => {
    cy.get(selector).then(subject => {
        cy.fixture(fileName).then(content => {
            const el = subject[0];
            const testFile = new File([content], fileName);
            const dataTransfer = new DataTransfer();

            dataTransfer.items.add(testFile);
            el.files = dataTransfer.files;
        });
    });
};

describe('Network page', () => {
    beforeEach(() => {
        cy.server();
        cy.fixture('network/networkGraph.json').as('networkGraphJson');
        cy.route('POST', api.network.networkGraph, '@networkGraphJson').as('networkGraph');
        cy.fixture('network/epoch.json').as('epochJson');
        cy.route('GET', api.network.epoch, '@epochJson').as('epoch');

        cy.visit(networkUrl);
        cy.wait('@networkGraph');
        cy.wait('@epoch');
    });

    it('should have selected item in nav bar', () => {
        cy.get(networkPageSelectors.network).click();
        cy.get(networkPageSelectors.network).should('have.class', 'bg-primary-700');
    });

    it('should display a legend', () => {
        cy.get(networkPageSelectors.legend.items)
            .eq(0)
            .should('have.attr', 'alt', 'deployment');
        cy.get(networkPageSelectors.legend.items)
            .eq(1)
            .should('have.attr', 'alt', 'deployment-allowed-connections');
        cy.get(networkPageSelectors.legend.items)
            .eq(2)
            .should('have.attr', 'alt', 'namespace');
        cy.get(networkPageSelectors.legend.items)
            .eq(3)
            .should('have.attr', 'alt', 'namespace-allowed-connection');
        cy.get(networkPageSelectors.legend.items)
            .eq(4)
            .should('have.attr', 'alt', 'namespace-connection');
        cy.get(networkPageSelectors.legend.items)
            .eq(5)
            .should('have.attr', 'alt', 'active-connection');
        cy.get(networkPageSelectors.legend.items)
            .eq(6)
            .should('have.attr', 'alt', 'allowed-connection');
        cy.get(networkPageSelectors.legend.items)
            .eq(7)
            .should('have.attr', 'alt', 'namespace-egress');
        cy.get(networkPageSelectors.legend.items)
            .eq(8)
            .should('have.attr', 'alt', 'namespace-ingress');
        cy.get(networkPageSelectors.legend.items)
            .eq(9)
            .should('have.attr', 'alt', 'namespace-egress-ingress');
    });

    it('should handle toggle click on simulator network policy button', () => {
        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        cy.get(networkPageSelectors.panels.simulatorPanel).should('be.visible');
        cy.get(networkPageSelectors.buttons.simulatorButtonOn).click();
        cy.get(networkPageSelectors.panels.simulatorPanel).should('not.be.visible');
    });

    it('should display error messages when uploaded wrong yaml', () => {
        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        uploadFile('network/policywithoutnamespace.yaml', 'input[type="file"]');
        cy.get(networkPageSelectors.simulatorSuccessMessage).should('not.be.visible');
    });

    it('should display success messages when uploaded right yaml', () => {
        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        uploadFile('network/policywithnamespace.yaml', 'input[type="file"]');
        cy.get(networkPageSelectors.simulatorSuccessMessage).should('be.visible');
    });
});
