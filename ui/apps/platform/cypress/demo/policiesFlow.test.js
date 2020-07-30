import { url, selectors as PoliciesSelectors } from '../constants/PoliciesPage';
import selectors from '../selectors/index';
import withAuth from '../helpers/basicAuth';

describe('Policies Flow', () => {
    withAuth();

    it('should take you to the Policies list from Platform Configuration side menu', () => {
        cy.visit('/');
        cy.get(selectors.navigation.leftNavBar).contains('Platform Configuration').click();
        cy.get(selectors.navigation.navPanel).contains('System Policies').click();
        cy.get(selectors.page.pageHeader).contains('Policies');
    });

    it('should close side panel when clicking "x"', () => {
        cy.visit(url);
        cy.get(selectors.table.rows).eq(0).click();
        cy.get(selectors.panel.sidePanel).should('exist');
        cy.wait(1000);
        cy.get(selectors.panel.closeButton).click();
        cy.get(selectors.panel.sidePanel).should('not.exist');
    });

    it('Selecting Edit should open policy edit panel', () => {
        cy.visit(url);
        cy.get(`${selectors.table.rows}:contains("Fixable CVSS >= 7")`).eq(0).click();
        cy.get(selectors.panel.sidePanel).should('exist');
        cy.get(selectors.panel.sidePanelHeader).contains('Fixable CVSS >= 7');
        cy.get(selectors.panel.editButton).click();
        cy.get(selectors.panel.form).should('exist');
    });

    it('Policy edit violated deployments list should contain expected deployments', () => {
        cy.visit(url);
        cy.get(`${selectors.table.rows}:contains("Fixable CVSS >= 7")`).eq(0).click();
        cy.get(selectors.panel.editButton).click();
        cy.get(selectors.panel.nextButton).click();
        cy.get(`${selectors.panel.sidePanel} ${selectors.table.rows}`).then((rows) => {
            expect(rows).to.have.length(44);
        });
    });

    it('Policy edit enforcement panel should show enforcement toggles', () => {
        cy.visit(url);
        cy.get(`${selectors.table.rows}:contains("Fixable CVSS >= 7")`).eq(0).click();
        cy.get(selectors.panel.editButton).click();
        cy.get(selectors.panel.nextButton).click();
        cy.wait(1000);
        cy.get(selectors.panel.nextButton).click();
        cy.get(PoliciesSelectors.enforcement.buildTile).should('exist');
        cy.get(PoliciesSelectors.enforcement.deployTile).should('exist');
    });

    it('Saving policy enforcement should update policy detail', () => {
        cy.visit(url);
        cy.get(`${selectors.table.rows}:contains("Fixable CVSS >= 7")`).eq(0).click();
        cy.get(selectors.panel.editButton).click();
        cy.get(selectors.panel.nextButton).click();
        cy.wait(1000);
        cy.get(selectors.panel.nextButton).click();
        cy.get(
            `${PoliciesSelectors.enforcement.buildTile} ${PoliciesSelectors.enforcement.onOffToggle} button:contains("On")`
        ).click();
        cy.get(
            `${PoliciesSelectors.enforcement.deployTile} ${PoliciesSelectors.enforcement.onOffToggle} button:contains("On")`
        ).click();
        cy.get(selectors.panel.saveButton).click();
        cy.get(`${selectors.panel.sidePanel} div:contains("Enforcement Action")`).should('exist');
    });

    it('should sort table alphabetically', () => {
        cy.visit(url);
        cy.get(selectors.table.column.name).click();
        cy.get(`${selectors.table.row.firstRow} [data-testid="policy-name"]`).contains(
            '30-Day Scan Age'
        );
        cy.get(selectors.table.column.name).click();
        cy.get(`${selectors.table.row.firstRow} [data-testid="policy-name"]`).contains(
            'Wget in Image'
        );
    });
});
