import { selectors } from './pages/PoliciesPage';

describe('Policies page', () => {
    beforeEach(() => {
        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click();
    });

    it('should have selected item in nav bar', () => {
        cy.get(selectors.configure).should('have.class', 'bg-primary-600');
    });

    it('should allow updating policy name', () => {
        const updatePolicyName = typeStr => {
            cy.get(selectors.editPolicyButton).click();
            cy.get('form input:first').type(typeStr);
            cy.get(selectors.nextButton).click();
            cy.get(selectors.savePolicyButton).click();
        };
        const secretSuffix = ':secretSuffix:';
        const deleteSuffix = '{backspace}'.repeat(secretSuffix.length);

        cy.get('table tr.cursor-pointer:first').click();
        updatePolicyName(secretSuffix);
        cy.get(`table tr td:contains("${secretSuffix}")`);
        updatePolicyName(deleteSuffix); // revert back
    });

    it('should open the preview panel to view policy dry run', () => {
        cy.get('table tr.cursor-pointer:first').click();
        cy.get(selectors.editPolicyButton).click();
        cy.get(selectors.nextButton).click();
        cy.get('.warn-message').should('exist');
        cy.get('.alert-preview').should('exist');
        cy.get('.whitelist-exclusions').should('exist');
        cy.get(selectors.cancelButton).click();
    });

    it('should open the panel to create a new policy', () => {
        cy.get(selectors.addPolicyButton).click();
    });
});
