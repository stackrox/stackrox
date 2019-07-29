import { selectors as RiskPageSelectors, url } from './constants/RiskPage';
import selectors from './constants/SearchPage';
import * as api from './constants/apiEndpoints';
import withAuth from './helpers/basicAuth';

describe('Risk page', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.fixture('risks/riskyDeployments.json').as('deploymentJson');
        cy.route('GET', api.risks.riskyDeployments, '@deploymentJson').as('deployments');

        cy.visit(url);
        cy.wait('@deployments');
    });

    const mockGetRisk = () => {
        cy.fixture('risks/firstDeploymentRisk.json').as('firstDeploymentRiskJson');
        cy.route('GET', api.risks.getRisk, '@firstDeploymentRiskJson').as('firstDeploymentRisk');
    };

    it('should have selected item in nav bar', () => {
        cy.get(RiskPageSelectors.risk).should('have.class', 'bg-primary-700');
    });

    it('should sort priority in the table', () => {
        cy.get(RiskPageSelectors.table.columns.priority).click({ force: true }); // ascending
        cy.get(RiskPageSelectors.table.columns.priority).click({ force: true }); // descending
        cy.get(RiskPageSelectors.table.row.firstRow).should('contain', '3');
    });

    it('should open the panel to view risk indicators', () => {
        mockGetRisk();
        cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
        cy.wait('@firstDeploymentRisk');
        cy.get(RiskPageSelectors.panelTabs.riskIndicators);
        cy.get(RiskPageSelectors.cancelButton).click();
    });

    it('should open the panel to view deployment details', () => {
        mockGetRisk();
        cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
        cy.wait('@firstDeploymentRisk');
        cy.get(RiskPageSelectors.panelTabs.deploymentDetails);
        cy.get(RiskPageSelectors.cancelButton).click();
    });

    it('should navigate from Risk Page to Images Page', () => {
        mockGetRisk();
        cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
        cy.wait('@firstDeploymentRisk');
        cy.get(RiskPageSelectors.panelTabs.deploymentDetails).click({ force: true });
        cy.get(RiskPageSelectors.imageLink)
            .first()
            .click({ force: true });
        cy.url().should('contain', '/main/images');
    });

    it('should close the side panel on search filter', () => {
        cy.get(selectors.pageSearchInput).type('Cluster:{enter}', { force: true });
        cy.get(selectors.pageSearchInput).type('remote{enter}', { force: true });
        cy.get(selectors.panelHeader)
            .eq(1)
            .should('not.be.visible');
    });

    it('should navigate to network page with selected deployment', () => {
        mockGetRisk();
        cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
        cy.wait('@firstDeploymentRisk');
        cy.get(RiskPageSelectors.networkNodeLink).click({ force: true });
        cy.url().should('contain', '/main/network');
    });
});
