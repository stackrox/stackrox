import { selectors } from './constants/TablePagination';
import withAuth from './helpers/basicAuth';

describe('Table pagination header in Policies page', () => {
    withAuth();

    it('should be visible', () => {
        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click({ force: true });
        cy.get(selectors.paginationHeader).should('be.visible');
    });

    it('should have previous page button disabled', () => {
        cy.get(selectors.prevPageButton).should('be.disabled');
    });

    // changing page size/row count from 20 to 50 breaks these tests --
    // will wait to re-enable once more policies are merged into master
    xit('should update page number', () => {
        cy.get(selectors.nextPageButton).click();
        cy.get(selectors.pageNumberInput).should('have.value', '2');
    });

    xit('should have last page next button disabled', () => {
        cy.get(selectors.nextPageButton).should('be.disabled');
    });

    xit('can update page number by typing in input', () => {
        cy.get(selectors.pageNumberInput)
            .clear()
            .invoke('attr', 'type', 'text'); // Cast
        cy.get(selectors.pageNumberInput).type('1');
        cy.get(selectors.tableFirstRow).should('contain', '30-Day Scan Age');
    });
});
