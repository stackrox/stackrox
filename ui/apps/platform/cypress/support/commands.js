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
            cy.injectAxe({ axeCorePath: Cypress.env('AXE_CORE_PATH') });
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

/**
 * This command installs a mock analytics object in the window object that can be used to assert
 * on the telemetry events that are emitted by the app. This mock installs and tracks events
 * regardless of whether telemetry is enabled or disabled in the system config.
 *
 * The mock analytics object is installed in the window object before ANY app scripts execute on
 * every new page load.
 *
 * The mock analytics object is also installed immediately on the current window (and cleared)
 * when the command is called.
 *
 * @returns {Cypress.Chainable} - A chainable object that can be used to get the telemetry events.
 */
Cypress.Commands.add('spyTelemetry', () => {
    function installMockAnalytics(win) {
        // eslint-disable-next-line no-param-reassign
        win.__cypress_telemetry_cache__ = { page: [], track: [] };

        // eslint-disable-next-line no-param-reassign
        win.analytics = {
            initialize: true,
            page: (type, name, properties) => {
                win.__cypress_telemetry_cache__.page.push({ type, name, properties });
            },
            track: (event, properties, context) => {
                win.__cypress_telemetry_cache__.track.push({ event, properties, context });
            },
        };
    }

    // Runs before ANY app scripts execute on every new page load
    Cypress.on('window:before:load', (win) => installMockAnalytics(win));

    // Also install immediately on the current window (and clear)
    return cy.window({ log: false }).then((win) => installMockAnalytics(win));
});

/**
 * This command returns the telemetry events that were emitted by the app. Note that this must
 * be called after the page has loaded and the telemetry events have been emitted.
 *
 * @returns {Cypress.Chainable} - A chainable object that can be used to get the telemetry page views and track events.
 */
Cypress.Commands.add('getTelemetryEvents', () => {
    return cy.window({ log: false }).its('__cypress_telemetry_cache__');
});
