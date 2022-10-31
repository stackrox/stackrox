// This function sets up auth headers for each test in the current test group.
export default () => {
    /*
     * Test isolation depends on experimentalSessionAndOrigin property in cypress.config.js file.
     *
     * First call in a test file set pages to about:blank
     * which clears cookies, local storage and session storage in all domains.
     * It caches session data set up in the callback function (that is, access token).
     *
     * Subsequent calls in a test file restore the cached session data.
     */
    cy.session('ROX_AUTH_TOKEN', () => {
        const token = Cypress.env('ROX_AUTH_TOKEN');
        if (token) {
            beforeEach(() => {
                localStorage.setItem('access_token', token);
            });
        }
    });
};
