import {
    url as environmentUrl,
    selectors as environmentPageSelectors
} from './constants/EnvironmentPage';
import * as api from './constants/apiEndpoints';

describe('Environment page', () => {
    beforeEach(() => {
        cy.server();
        cy.fixture('environment/networkGraph.json').as('networkGraphJson');
        cy.route('GET', api.environment.networkGraph, '@networkGraphJson').as('networkGraph');
        cy.fixture('environment/epoch.json').as('epochJson');
        cy.route('GET', api.environment.epoch, '@epochJson').as('epoch');

        cy.visit(environmentUrl);
        cy.wait('@networkGraph');
        cy.wait('@epoch');
    });

    it('should display a Legend', () => {
        cy.get(environmentPageSelectors.legend.deployment).contains('Deployment');
        cy.get(environmentPageSelectors.legend.namespace).contains('Namespace');
        cy.get(environmentPageSelectors.legend.ingressEgress).contains('Ingress/Egress');
        cy.get(environmentPageSelectors.legend.internetEgress).contains('Internet Egress');
    });

    it('should have 3 namespaces in total', () => {
        cy.get(environmentPageSelectors.namespaces.all).should('have.length', 3);
    });

    it('should have 4 services in total', () => {
        cy.get(environmentPageSelectors.services.all).should('have.length', 4);
    });

    it('should have 1 service link', () => {
        cy.get(environmentPageSelectors.links.services).should('have.length', 1);
    });

    it('should have 2 namespace links', () => {
        cy.get(environmentPageSelectors.links.namespaces).should('have.length', 2);
    });

    it('should have 2 bidirectional links', () => {
        cy.get(environmentPageSelectors.links.bidirectional).should('have.length', 2);
    });
});
