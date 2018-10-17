import { url as loginUrl, selectors } from './constants/LoginPage';
import { url as complianceUrl } from './constants/CompliancePage';

import * as api from './constants/apiEndpoints';

describe('Authentication', () => {
    const setupAuth = (landingUrl, authStatusValid = true) => {
        cy.server();
        cy.route('GET', api.auth.authProviders, 'fixture:auth/authProviders.json').as(
            'authProviders'
        );
        cy.route({
            method: 'GET',
            url: api.auth.authStatus,
            status: authStatusValid ? 200 : 401,
            response: {}
        }).as('authStatus');

        cy.visit(landingUrl);
        cy.wait('@authProviders');
    };

    it('should redirect user to login page, authenticate and redirect to the requested page', () => {
        setupAuth(complianceUrl);
        cy.url().should('contain', loginUrl);
        cy.get(selectors.providerSelect).should('have.text', 'auth-provider-name');
        cy.get(selectors.loginButton).click(); // stubbed auth provider will simulate redirect with 'my-token'
        cy.wait('@authStatus').then(xhr => {
            expect(xhr.request.headers.Authorization).to.eq('Bearer my-token');
        });
        cy.url().should('contain', complianceUrl);
    });

    it('should allow authenticated user to enter', () => {
        localStorage.setItem('access_token', 'my-token'); // simulate authenticated user
        setupAuth(complianceUrl);
        cy.url().should('contain', complianceUrl);
    });

    it('should logout previously authenticated user with invalid token', () => {
        localStorage.setItem('access_token', 'my-token'); // invalid token
        setupAuth(complianceUrl, false);
        cy.url().should('contain', loginUrl);
    });

    it('should logout user by request', () => {
        localStorage.setItem('access_token', 'my-token'); // authenticated user
        setupAuth(complianceUrl);
        cy.get('button:contains("Logout")').click();
        cy.url().should('contain', loginUrl);
    });

    it('should retry when token has changed after the request was made', () => {
        /**
         * Test case is inspired by https://stack-rox.atlassian.net/browse/ROX-397.
         * The idea of the test is to cover this scenario
         *   1. Request is made to the server with token1 (which can be no token)
         *   2. Before the response, another browser tab changes token in local storage (e.g. user logs in)
         *   3. Response comes back with 401, but token changed, so it should retry with a new token
         */
        localStorage.setItem('access_token', 'first-token');
        cy.server();
        cy.route({
            method: 'GET',
            url: api.benchmarks.configs,
            status: 401,
            delay: 200,
            response: {},
            onRequest: () => {
                localStorage.setItem('access_token', 'new-token');
            }
        }).as('benchmarks');
        setupAuth(complianceUrl);

        cy.wait('@benchmarks').then(xhr => {
            expect(xhr.request.headers.Authorization).to.eq('Bearer first-token');
        });
        // should retry request with a new token
        cy.wait('@benchmarks').then(xhr => {
            expect(xhr.request.headers.Authorization).to.eq('Bearer new-token');
        });
    });
});
