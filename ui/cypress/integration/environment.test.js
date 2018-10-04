import { url as networkUrl, selectors as networkPageSelectors } from './constants/EnvironmentPage';
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
        cy.fixture('environment/networkGraph.json').as('networkGraphJson');
        cy.route('POST', api.environment.networkGraph, '@networkGraphJson').as('networkGraph');
        cy.fixture('environment/epoch.json').as('epochJson');
        cy.route('GET', api.environment.epoch, '@epochJson').as('epoch');

        cy.visit(networkUrl);
        cy.wait('@networkGraph');
        cy.wait('@epoch');
    });

    it('should have selected item in nav bar', () => {
        cy.get(networkPageSelectors.network).click();
        cy.get(networkPageSelectors.network).should('have.class', 'bg-primary-700');
    });

    it('should display a Legend', () => {
        cy.get(networkPageSelectors.legend.deployment).contains('Deployment');
        cy.get(networkPageSelectors.legend.namespace).contains('Namespace');
        cy.get(networkPageSelectors.legend.ingressEgress).contains('Ingress/Egress');
        cy.get(networkPageSelectors.legend.internetEgress).contains('Internet Egress');
    });

    it('should have 3 namespaces in total', () => {
        cy.get(networkPageSelectors.namespaces.all).should('have.length', 3);
    });

    it('should have 4 services in total', () => {
        cy.get(networkPageSelectors.services.all).should('have.length', 4);
    });

    it('should have 1 service link', () => {
        cy.get(networkPageSelectors.links.services).should('have.length', 1);
    });

    it('should have 2 namespace links', () => {
        cy.get(networkPageSelectors.links.namespaces).should('have.length', 2);
    });

    it('should have 2 bidirectional links', () => {
        cy.get(networkPageSelectors.links.bidirectional).should('have.length', 2);
    });

    it('should handle toggle click on simulator network policy button', () => {
        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        cy.get(networkPageSelectors.panels.simulatorPanel).should('be.visible');
        cy.get(networkPageSelectors.buttons.simulatorButtonOn).click();
        cy.get(networkPageSelectors.panels.simulatorPanel).should('not.be.visible');
    });

    it('should display error messages when uploaded wrong yaml', () => {
        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        uploadFile('environment/policywithoutnamespace.yaml', 'input[type="file"]');
        cy.get(networkPageSelectors.simulatorSuccessMessage).should('not.be.visible');
    });

    it('should display success messages when uploaded right yaml', () => {
        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        uploadFile('environment/policywithnamespace.yaml', 'input[type="file"]');
        cy.get(networkPageSelectors.simulatorSuccessMessage).should('be.visible');
    });
});
