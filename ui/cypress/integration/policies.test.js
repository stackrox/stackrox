import { url as policiesUrl, selectors } from './pages/PoliciesPage';

describe('Policies page', () => {
    beforeEach(() => {
        cy.visit(policiesUrl);
    });

    it('should have selected item in nav bar', () => {
        cy.get('nav li:contains("Policies") a').should('have.class', 'bg-primary-600');
    });

    it('should allow updating policy name', () => {
        const updatePolicyName = typeStr => {
            cy.get(selectors.editPolicyButton).click();
            cy.get('form input:first').type(typeStr);
            cy.get(selectors.savePolicyButton).click();
        };
        const secretSuffix = ':secretSuffix:';
        const deleteSuffix = '{backspace}'.repeat(secretSuffix.length);

        cy.get('table tr.cursor-pointer:first').click();
        updatePolicyName(secretSuffix);
        cy.get(`table tr td:contains("${secretSuffix}")`);
        updatePolicyName(deleteSuffix); // revert back
    });

    it('should open the panel to create a new policy', () => {
        cy.get(selectors.addPolicyButton).click();
    });
});
