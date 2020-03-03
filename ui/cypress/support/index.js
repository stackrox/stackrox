import './commands';

if (Cypress.env('COLLECT_COVERAGE')) {
    // eslint-disable-next-line global-require
    require('@cypress/code-coverage/support');
}
