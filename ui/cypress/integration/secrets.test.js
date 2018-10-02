import { selectors as SecretsPageSelectors } from './constants/SecretsPage';

describe('Secrets page', () => {
    beforeEach(() => {
        cy.visit('/');
        cy.get(SecretsPageSelectors.secrets).click();
    });

    it('should have selected item in nav bar', () => {
        cy.get(SecretsPageSelectors.secrets).should('have.class', 'bg-primary-700');
    });

    it('should open the panel to view secret details', () => {
        cy
            .get(SecretsPageSelectors.table.rows)
            .first()
            .click();
        cy.get(SecretsPageSelectors.panel.secretDetails);
        cy.get(SecretsPageSelectors.cancelButton).click();
    });

    it('should navigate from Secrets Page to Risk Page', () => {
        cy
            .get(SecretsPageSelectors.table.rows)
            .first()
            .click();
        cy
            .get(SecretsPageSelectors.deploymentLinks)
            .first()
            .click();
        cy.url().should('contain', '/main/risk');
    });

    it('should display a search input with only the cluster search modifier', () => {
        cy.get(SecretsPageSelectors.searchInput).type('Cluster:{enter}', { force: true });
        cy.get(SecretsPageSelectors.searchInput).type('remote{enter}', { force: true });
    });
});
