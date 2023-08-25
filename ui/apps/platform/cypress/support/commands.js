Cypress.Commands.add('getCytoscape', (containerId) => {
    cy.wait(100);
    cy.get(containerId).then(() => {
        cy.window().then((win) => {
            return win.cytoscape;
        });
    });
});
