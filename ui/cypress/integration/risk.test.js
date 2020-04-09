import { selectors as RiskPageSelectors, url, errorMessages } from '../constants/RiskPage';
import selectors from '../constants/SearchPage';
import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';

describe('Risk page', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.fixture('risks/riskyDeployments.json').as('deploymentJson');
        cy.route('GET', api.risks.riskyDeployments, '@deploymentJson').as('deployments');

        cy.visit(url);
        cy.wait('@deployments');
    });

    const mockGetDeployment = () => {
        cy.fixture('risks/firstDeployment.json').as('firstDeploymentJson');
        cy.route('GET', api.risks.getDeploymentWithRisk, '@firstDeploymentJson').as(
            'firstDeployment'
        );
    };

    it('should have selected item in nav bar', () => {
        cy.get(RiskPageSelectors.risk).should('have.class', 'bg-primary-700');
    });

    it('should sort priority in the table', () => {
        cy.get(RiskPageSelectors.table.column.priority).click({ force: true }); // ascending
        cy.get(RiskPageSelectors.table.column.priority).click({ force: true }); // descending
        cy.get(RiskPageSelectors.table.row.firstRow).should('contain', '3');
    });

    it('should highlight selected deployment row', () => {
        cy.get(RiskPageSelectors.table.row.firstRow)
            .click({ force: true })
            .should('have.class', 'row-active');
    });

    it('should display deployment error message in panel', () => {
        cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
        cy.get(RiskPageSelectors.errMgBox).contains(errorMessages.deploymentNotFound);
    });

    it('should display error message in process discovery tab', () => {
        mockGetDeployment();
        cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
        cy.wait('@firstDeployment');

        cy.get(RiskPageSelectors.panelTabs.processDiscovery).click();
        cy.get(RiskPageSelectors.errMgBox).contains(errorMessages.processNotFound);
        cy.get(RiskPageSelectors.cancelButton).click();
    });

    it('should open the panel to view risk indicators', () => {
        mockGetDeployment();
        cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
        cy.wait('@firstDeployment');

        cy.get(RiskPageSelectors.panelTabs.riskIndicators);
        cy.get(RiskPageSelectors.cancelButton).click();
    });

    it('should open the panel to view deployment details', () => {
        mockGetDeployment();
        cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
        cy.wait('@firstDeployment');

        cy.get(RiskPageSelectors.panelTabs.deploymentDetails);
        cy.get(RiskPageSelectors.cancelButton).click();
    });

    it('should navigate from Risk Page to Vulnerability Management Image Page', () => {
        mockGetDeployment();
        cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
        cy.wait('@firstDeployment');

        cy.get(RiskPageSelectors.panelTabs.deploymentDetails).click({ force: true });
        cy.get(RiskPageSelectors.imageLink)
            .first()
            .click({ force: true });
        cy.url().should('contain', '/main/vulnerability-management/image');
    });

    it('should close the side panel on search filter', () => {
        cy.get(selectors.pageSearchInput).type('Cluster:{enter}', { force: true });
        cy.get(selectors.pageSearchInput).type('remote{enter}', { force: true });
        cy.get(selectors.panelHeader)
            .eq(1)
            .should('not.be.visible');
    });

    it('should navigate to network page with selected deployment', () => {
        mockGetDeployment();
        cy.get(RiskPageSelectors.table.row.firstRow).click({ force: true });
        cy.wait('@firstDeployment');

        cy.get(RiskPageSelectors.networkNodeLink).click({ force: true });
        cy.url().should('contain', '/main/network');
    });
});
