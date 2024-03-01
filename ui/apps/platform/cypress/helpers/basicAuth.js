// This function sets up auth headers for each test in the current test group.
export default () => {
    const token = Cypress.env('ROX_AUTH_TOKEN');
    if (token) {
        beforeEach(() => {
            cy.intercept('*', (req) => {
                req.headers.Authorization = `Bearer ${token}`;
            });
        });
    }
};
