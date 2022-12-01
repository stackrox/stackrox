import addSeconds from 'date-fns/add_seconds';

import { url as loginUrl, selectors } from '../constants/LoginPage';

import * as api from '../constants/apiEndpoints';

const pagePath = '/main/systemconfig';

const AUTHENTICATED = true;
const UNAUTHENTICATED = false;

describe('Authentication', () => {
    const setupAuth = (landingUrl, authStatusValid, authStatusResponse = {}) => {
        cy.intercept('GET', api.auth.loginAuthProviders, { fixture: 'auth/authProviders.json' }).as(
            'authProviders'
        );
        cy.intercept('GET', api.auth.authStatus, {
            statusCode: authStatusValid ? 200 : 401,
            body: authStatusResponse,
        }).as('authStatus');

        cy.visit(landingUrl);
        cy.wait('@authProviders');
    };

    const stubAPIs = () => {
        // Cypress routes have an override behaviour, so defining this first makes it the fallback.
        // Replace /.*/ RegExp for route method with '/v1/*' string for intercept method
        // because it is not limited to XHR, therefore it matches HTML requests too!
        cy.intercept('/v1/*', { body: {} }).as('everythingElse');
    };

    it('should redirect user to login page, authenticate and redirect to the requested page', () => {
        stubAPIs();
        localStorage.setItem('access_token', 'my-token'); // replace possible valid token left over from previous test file
        setupAuth(pagePath, AUTHENTICATED);
        cy.location('pathname').should('eq', loginUrl);
        cy.get(selectors.providerSelect).should('have.text', 'auth-provider-name');
        cy.get(selectors.loginButton).click(); // stubbed auth provider will simulate redirect with 'my-token'
        // Replace Authorization for route method with authorization for intercept method.
        cy.wait('@authStatus').its('request.headers.authorization').should('eq', 'Bearer my-token');
        cy.location('pathname').should('eq', pagePath);
    });

    it('should allow authenticated user to enter', () => {
        stubAPIs();
        localStorage.setItem('access_token', 'my-token'); // simulate authenticated user
        setupAuth(pagePath, AUTHENTICATED);

        cy.wait('@authStatus');

        cy.location('pathname').should('eq', pagePath);
    });

    it('should logout previously authenticated user with invalid token', () => {
        stubAPIs();
        localStorage.setItem('access_token', 'my-token'); // invalid token
        setupAuth(pagePath, UNAUTHENTICATED);

        cy.wait('@authStatus');

        cy.location('pathname').should('eq', loginUrl);
    });

    // TODO: Fix it, see ROX-4983 for more explanation
    it.skip('should request token refresh 30 sec in advance', () => {
        stubAPIs();
        cy.intercept('POST', api.auth.tokenRefresh, { body: {} }).as('tokenRefresh');
        localStorage.setItem('access_token', 'my-token'); // authenticated user

        const expiryDate = addSeconds(Date.now(), 33); // +3 sec should be enough
        setupAuth(pagePath, AUTHENTICATED, {
            expires: expiryDate.toISOString(),
        });

        cy.wait('@tokenRefresh');
    });
});
