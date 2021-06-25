// eslint-disable-next-line import/prefer-default-export
export function editIntegration(name) {
    cy.get(`tr:contains('${name}') td.pf-c-table__action button`).click();
    cy.get(
        `tr:contains('${name}') td.pf-c-table__action button:contains("Edit Integration")`
    ).click();
}
