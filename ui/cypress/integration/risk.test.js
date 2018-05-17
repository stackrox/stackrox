import { selectors as RiskPageSelectors } from './pages/RiskPage';
import selectors from './pages/SearchPage';

describe('Risk page', () => {
    beforeEach(() => {
        cy.visit('/');
        cy.get(RiskPageSelectors.risk).click();
    });

    it('should have selected item in nav bar', () => {
        cy.get(RiskPageSelectors.risk).should('have.class', 'bg-primary-600');
    });

    it('should open the panel to view risk indicators', () => {
        cy.get('table tr.cursor-pointer:first').click();
        cy
            .get(RiskPageSelectors.panelTabs.riskIndicators)
            .first()
            .should('have.class', 'tab-active');
        cy.get(RiskPageSelectors.cancelButton).click();
    });

    it('should show mounts in deployment details tab', () => {
        cy.get('table tr.cursor-pointer:first').click();
        cy.get(RiskPageSelectors.panelTabs.deploymentDetails).click();
        cy.get(RiskPageSelectors.mounts.label).should('be.visible');
        cy.get(RiskPageSelectors.mounts.items).should('have.length', 2);
    });

    it('should navigate from Risk Page to Images Page', () => {
        cy.get('table tr.cursor-pointer:first').click();
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
