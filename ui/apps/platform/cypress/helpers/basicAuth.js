// This function sets up auth headers for each test in the current test group.
export default () => {
    beforeEach(() => {
        /*
         * Test isolation depends on experimentalSessionAndOrigin property in cypress.config.js file.
         *
         * First call in a test file set pages to about:blank
         * which clears cookies, local storage and session storage in all domains.
         *
         * Subsequent calls in a test file restore the cached session data.
         */
        cy.session('ROX_AUTH_TOKEN', () => {});

        // Do not include auth token in cached session data, Because CI refreshes it periodically.
        const token = Cypress.env('ROX_AUTH_TOKEN');
        if (token) {
            localStorage.setItem('access_token', token);
        }
    });
};
