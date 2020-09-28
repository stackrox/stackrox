// adds "upload" command, see https://github.com/abramenal/cypress-file-upload
import 'cypress-file-upload';

// adds "cytoscape" command for accessing network graph nodes and edges
Cypress.Commands.add('cytoscape', (containerId) => {
    cy.get(containerId).then(() => {
        cy.window().then((win) => {
            return win.cytoscape;
        });
    });
});
