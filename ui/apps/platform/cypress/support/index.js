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
Cypress.on(
    'uncaught:exception',
    (err) => !err.message.includes('ResizeObserver loop limit exceeded')
);
