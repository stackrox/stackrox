import { selectors } from './pages/RiskPage';

describe('Risk page', () => {
    beforeEach(() => {
        cy.visit('/');
        cy.get(selectors.risk).click();
    });

    it('should have selected item in nav bar', () => {
        cy.get(selectors.risk).should('have.class', 'bg-primary-600');
    });

    it('should open the panel to view risk indicators', () => {
        cy.get('table tr.cursor-pointer:first').click();
        cy.get(selectors.panelTabs.riskIndicators).should('have.class', 'tab-active');
        cy.get(selectors.cancelButton).click();
    });

    it('should show mounts in deployment details tab', () => {
        cy.get('table tr.cursor-pointer:first').click();
        cy.get(selectors.panelTabs.deploymentDetails).click();
        cy.get(selectors.mounts.label).should('be.visible');
        cy.get(selectors.mounts.items).should('have.length', 2);
    });

    it('should navigate from Risk Page to Images Page', () => {
        cy.get('table tr.cursor-pointer:first').click();
        cy.get(selectors.panelTabs.deploymentDetails).click();
        cy
            .get(selectors.imageLink)
            .first()
            .click();
        cy.url().should('contain', '/main/images');
    });
});
