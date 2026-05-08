// This function sets up auth headers for each test in the current test group.
export default () => {
    beforeEach(() => {
        cy.env(['ROX_AUTH_TOKEN']).then(({ ROX_AUTH_TOKEN }) => {
            if (ROX_AUTH_TOKEN) {
                localStorage.setItem('access_token', ROX_AUTH_TOKEN);
            } else {
                cy.log('WARNING: ROX_AUTH_TOKEN is not set, tests will run unauthenticated');
            }
        });
    });
};
