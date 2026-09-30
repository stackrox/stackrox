import { isBenignUncaughtException } from '../helpers/uncaughtExceptions';

import './commands';

// fix a long-standing problem in Cypress where elements that are otherwise clickable
//   get scrolled out of view, and thus cause false positives
//   See: https://github.com/cypress-io/cypress/issues/871,
//        and the solution, later in that thread
//        https://github.com/cypress-io/cypress/issues/871#issuecomment-509392310
Cypress.on('scrolled', ($el) => {
    $el.get(0).scrollIntoView({
        block: 'center',
        inline: 'center',
    });
});

// Ignore uncaught exceptions that are benign in test execution, such as the Chrome-specific
// ResizeObserver error. See isBenignUncaughtException for the list and the reason for each.
Cypress.on('uncaught:exception', (err) => !isBenignUncaughtException(err));

// Output timestamps for test suite start and end
before(() => {
    cy.task('beforeSuite', Cypress.spec);
});

// Output timestamp at the point of test failure
Cypress.on('fail', (err) => {
    // eslint-disable-next-line no-param-reassign
    err.name = `${err.name} - ${new Date().toISOString()}`;
    throw err;
});
