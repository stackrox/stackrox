import pf6 from '../selectors/pf6';

export function selectProject(project: string) {
    cy.get(`.co-namespace-bar ${pf6.menuToggle}`).click();
    cy.get('.co-namespace-bar input[aria-label="Select project..."]').clear().type(project);
    cy.get(`.co-namespace-bar ${pf6.menu}`)
        .contains('[role="menuitem"]', new RegExp(`^${project}$`, 'i'))
        .click();
}

export function filterByField(field: string, value: string) {
    cy.get(`[data-ouia-component-id="DataViewFilters"] ${pf6.menuToggle}`).first().click();
    cy.get(pf6.menuItem).contains(field).click();
    // case insensitive search for the field + "filter" handles both 'Name filter' and 'Filter by name' forms
    // that are used across console versions
    cy.get(`input[aria-label*="${field}" i][aria-label*="filter" i]`).should('not.be.disabled');
    cy.get(`input[aria-label*="${field}" i][aria-label*="filter" i]`).type(value);
}
