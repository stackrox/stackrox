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

// Fixes another long standing problem in Cypress where a Chrome specific error
// is thrown due to multiple firings of a ResizeObserver that is benign in test execution.
// There are multiple closed PRs to add this to Cypress core that were declined in favor
// of recommending users add the following event handler to this `support` file instead.
//
//  See various Cypress threads:
//  - PR to add in Cypress core: https://github.com/cypress-io/cypress/pull/20257 (closed)
//  - PR to add in Cypress core: https://github.com/cypress-io/cypress/pull/20284 (closed)
//  - Comment with the original fix recommendation: https://github.com/cypress-io/cypress/issues/8418#issuecomment-992564877
//
//
// Addendum (2023-03-27): ignore error about missing language function
// This is from the PatternFly code editor, related to our adding the YAML language module to it
//
Cypress.on(
    'uncaught:exception',
    (err) =>
        !err.message.includes('ResizeObserver loop completed') &&
        !err.message.includes('ResizeObserver loop limit exceeded') &&
        // Addendum (2022-11-28): ignore error about multiple versions of Mobx
        // The patternfly topology extension uses a 5.x version of Mobx, and the redoc library needs a 6.x version
        //
        // TODO: remove this catch for multiple MobX versions after the in-product documentation is removed
        !err.message.includes('There are multiple, different versions of MobX active') &&
        // Addendum (2023-03-27): ignore error related to PatternFly code editor
        !err.message.includes('model.getLanguageId is not a function') &&
        !err.message.includes("Uncaught SyntaxError: Unexpected token '<'") &&
        !err.message.includes("Uncaught SyntaxError: Unexpected token '<'")
);
