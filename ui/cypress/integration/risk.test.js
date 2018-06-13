import { selectors as RiskPageSelectors } from './constants/RiskPage';
import selectors from './constants/SearchPage';

describe('Risk page', () => {
    beforeEach(() => {
        cy.visit('/');
        cy.get(RiskPageSelectors.risk).click();
    });

    it('should have selected item in nav bar', () => {
        cy.get(RiskPageSelectors.risk).should('have.class', 'bg-primary-600');
    });

    it('should open the panel to view risk indicators', () => {
        cy.get(RiskPageSelectors.table.row.prevent_sensor).click();
        cy
            .get(RiskPageSelectors.panelTabs.riskIndicators)
            .first()
            .should('have.class', 'tab-active');
        cy.get(RiskPageSelectors.cancelButton).click();
    });

    it('should open the panel to view deployment details', () => {
        cy.get(RiskPageSelectors.table.row.prevent_sensor).click();
        cy.get(RiskPageSelectors.panelTabs.deploymentDetails);
        cy.get(RiskPageSelectors.cancelButton).click();
    });

    it('should navigate from Risk Page to Images Page', () => {
        cy.get(RiskPageSelectors.table.row.prevent_sensor).click();
        cy.get(RiskPageSelectors.panelTabs.deploymentDetails).click();
        cy
            .get(RiskPageSelectors.imageLink)
            .first()
            .click();
        cy.url().should('contain', '/main/images');
    });

    it('should close the side panel on search filter', () => {
        cy.get(selectors.pageSearchInput).type('Cluster:{enter}', { force: true });
        cy.get(selectors.pageSearchInput).type('remote{enter}', { force: true });
        cy.get('.side-panel').should('not.be.visible');
    });
});
