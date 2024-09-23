import '@testing-library/cypress/add-commands';
import 'cypress-axe';

/**
 * Custom Cypress command to check the accessibility of the page or a specific element.
 * This command injects the Axe accessibility testing library into the page if it hasn't been injected yet,
 * and then runs an accessibility scan on the specified context.
 *
 * @param {string | JQuery<HTMLElement> | HTMLElement | null} [context=null] - The context to run the accessibility check on.
 *   - `null`: Runs the accessibility check on the full page.
 *   - `string`: A CSS selector to define the context (e.g., '#main-content').
 *   - `JQuery<HTMLElement>` or `HTMLElement`: A DOM element or a jQuery-wrapped element to run the check on.
 */

Cypress.Commands.add('checkAccessibility', (context = null) => {
    // inject axe only if it hasn't been injected yet
    cy.window().then((win) => {
        if (!win.axe) {
            cy.injectAxe();
        }
    });

    cy.checkA11y(context, null, (violations) => {
        violations.forEach((violation) => {
            const nodes = Cypress.$(violation.nodes.map((node) => node.target).join(','));
            Cypress.log({
                name: `Accessibility Violation - ${violation.impact}`,
                consoleProps: () => violation,
                $el: nodes,
                message: `[${violation.help}](${violation.helpUrl})`,
            });
            nodes.css('outline', '2px solid red');
        });
    });
});
