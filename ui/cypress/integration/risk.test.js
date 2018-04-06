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
        cy
            .get(selectors.panelTabs)
            .first()
            .should('have.class', 'tab-active');
        cy.get(selectors.cancelButton).click();
    });
});
