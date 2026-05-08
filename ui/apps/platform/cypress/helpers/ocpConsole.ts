import pf6 from '../selectors/pf6';

export function selectProject(project: string) {
    cy.get(`.co-namespace-bar ${pf6.menuToggle}`).click();
    cy.get('.co-namespace-bar input[aria-label="Select project..."]').clear().type(project);
    cy.get(`.co-namespace-bar ${pf6.menu}`)
        .contains('[role="menuitem"]', new RegExp(`^${project}$`, 'i'))
        .click();
}

export function filterByField(field: string, value: string) {
    // OCP 4.19 uses a PF6 Dropdown (data-test-id), 4.21+ uses a DataViewFilters MenuToggle (OUIA)
    cy.get(
        `[data-test-id="dropdown-button"], [data-ouia-component-id="DataViewFilters"] ${pf6.menuToggle}`
    )
        .first()
        .click();
    cy.contains('[data-test-id="dropdown-menu"], [role="menuitem"]', field).first().click();

    // OCP 4.19 uses "Search by"  + field name, 4.21+ uses "Filter by" + field name
    cy.get(`input[aria-label*="Search by ${field}" i], input[aria-label*="Filter by ${field}" i]`)
        .first()
        .should('not.be.disabled');
    cy.get(`input[aria-label*="Search by ${field}" i], input[aria-label*="Filter by ${field}" i]`)
        .first()
        .type(value, { delay: 0 });
}
