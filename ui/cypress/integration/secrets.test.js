import { selectors as SecretsPageSelectors, url as secretsUrl } from './constants/SecretsPage';
import withAuth from './helpers/basicAuth';

describe('Secrets page', () => {
    withAuth();

    beforeEach(() => {
        cy.visit(secretsUrl);
    });

    it('should have selected item in nav bar', () => {
        cy.get(SecretsPageSelectors.secrets).should('have.class', 'bg-primary-700');
    });

    it('should open the panel to view secret details', () => {
        // TODO: need to force the click because there is probably some issue with Electron / Cypress / product UI
        // that somehow table on it's appearance is scrolled down and first row is hidden behind the table header.
        cy.get(SecretsPageSelectors.table.firstRow).click({ force: true });
        cy.get(SecretsPageSelectors.panel.secretDetails);
        cy.get(SecretsPageSelectors.cancelButton).click();
    });

    it('should navigate from Secrets Page to Risk Page', () => {
        // TODO: need to force the click because there is probably some issue with Electron / Cypress / product UI
        // that somehow table on it's appearance is scrolled down and first row is hidden behind the table header.
        cy.get(SecretsPageSelectors.table.firstRow).click({ force: true });
        cy.get(SecretsPageSelectors.deploymentLinks)
            .first()
            .click();
        cy.url().should('contain', '/main/risk');
    });

    it('should display a search input with only the cluster search modifier', () => {
        cy.get(SecretsPageSelectors.searchInput).type('Cluster:{enter}', { force: true });
        cy.get(SecretsPageSelectors.searchInput).type('remote{enter}', { force: true });
    });
});
