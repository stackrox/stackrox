// adds "upload" command, see https://github.com/abramenal/cypress-file-upload
import 'cypress-file-upload';

Cypress.Commands.add('getCytoscape', (containerId) => {
    cy.get(containerId).then(() => {
        cy.window().then((win) => {
            return win.cytoscape;
        });
    });
});
