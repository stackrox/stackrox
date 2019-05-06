import { url, selectors } from './constants/CompliancePage';
import withAuth from './helpers/basicAuth';

describe('Compliance list page', () => {
    withAuth();

    it('should open/close side panel when clicking on a table row', () => {
        cy.visit(url.list.clusters);
        cy.get(selectors.list.table.firstRowName)
            .invoke('text')
            .then(name => {
                cy.get(selectors.list.table.firstRow).click();
                cy.get(selectors.list.panels)
                    .its('length')
                    .should('eq', 2);
                cy.get(selectors.list.sidePanelHeader).contains(name);
                cy.get(selectors.widget.relatedEntities).should('not.exist');
                cy.get(selectors.list.sidePanelCloseBtn).click();
                cy.get(selectors.list.panels)
                    .its('length')
                    .should('eq', 1);
            });
    });

    it('should link to entity page when clicking on side panel header', () => {
        cy.visit(url.list.clusters);
        cy.get(selectors.list.table.firstRow).click();
        cy.get(selectors.list.sidePanelHeader).click();
        cy.url().should('include', url.list.clusters);
    });

    it('should collapse/open table banner', () => {
        cy.visit(url.list.clusters);
        cy.get(selectors.list.banner.content).should('be.visible');
        cy.get(selectors.list.banner.collapseButton).click();
        cy.get(selectors.list.banner.content).should('be.not.visible');
    });
});
