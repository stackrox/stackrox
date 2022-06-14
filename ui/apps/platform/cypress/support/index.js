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
