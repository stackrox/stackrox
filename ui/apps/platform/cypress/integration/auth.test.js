import addSeconds from 'date-fns/add_seconds';

import * as api from '../constants/apiEndpoints';
import { url as loginUrl, selectors } from '../constants/LoginPage';

const pagePath = '/main/systemconfig';

// Authentication providers

const loginAuthProvidersAlias = 'login/authproviders';

function visitAndWaitForAuthProviders(destinationUrl) {
    const fixture = 'auth/authProviders.json';
    cy.intercept('GET', api.auth.loginAuthProviders, { fixture }).as(loginAuthProvidersAlias);

    cy.visit(destinationUrl);

    cy.wait(`@${loginAuthProvidersAlias}`);
}

// Authentication status

const authStatusAlias = 'auth/status';

function interactAndWaitForAuthStatus(interactionCallback, staticResponseForAuthStatus) {
    cy.intercept('GET', api.auth.authStatus, staticResponseForAuthStatus).as(authStatusAlias);

    interactionCallback();

    return cy.wait(`@${authStatusAlias}`);
}

// System Configuration

const systemConfigAlias = 'config';

function reachSystemConfiguration(interactionCallback) {
    cy.intercept('GET', '/v1/config').as(systemConfigAlias);

    interactionCallback();

    cy.location('pathname').should('eq', pagePath);
    cy.wait(`@${systemConfigAlias}`);
    cy.get('h1:contains("System Configuration")');
}

describe('Authentication', () => {
    // Intentionally omit withAuth() call.

    it('should redirect user to login page, authenticate and redirect to the requested page', () => {
        const staticResponseForAuthStatusOK = {
            statusCode: 200,
            body: {},
        };

        visitAndWaitForAuthProviders(pagePath);

        cy.location('pathname').should('eq', loginUrl);
        // Assertion corresponds to value of name property in fixture for visit function call above.
        cy.get(selectors.providerSelect).should('have.text', 'auth-provider-name');

        reachSystemConfiguration(() => {
            interactAndWaitForAuthStatus(() => {
                cy.get(selectors.loginButton).click();
            }, staticResponseForAuthStatusOK)
                .its('request.headers.authorization')
                .should('eq', 'Bearer my-token'); // assertion corresponds to token=my-token in fixture
        });
    });

    it('should allow authenticated user to enter', () => {
        const staticResponseForAuthStatusOK = {
            statusCode: 200,
            body: {},
        };

        localStorage.setItem('access_token', 'my-token'); // authenticated user

        reachSystemConfiguration(() => {
            interactAndWaitForAuthStatus(() => {
                visitAndWaitForAuthProviders(pagePath);
            }, staticResponseForAuthStatusOK);
        });
    });

    it('should logout previously authenticated user with invalid token', () => {
        const staticResponseForAuthStatusUnauthorized = {
            statusCode: 401,
            body: {},
        };

        localStorage.setItem('access_token', 'my-token'); // invalid token

        interactAndWaitForAuthStatus(() => {
            visitAndWaitForAuthProviders(pagePath);
        }, staticResponseForAuthStatusUnauthorized);

        cy.location('pathname').should('eq', loginUrl);
    });

    // TODO: Fix it, see ROX-4983 for more explanation
    it.skip('should request token refresh 30 sec in advance', () => {
        const staticResponseForAuthStatusOK = {
            statusCode: 200,
            body: {
                expires: addSeconds(Date.now(), 33).toISOString(), // +3 sec should be enough
            },
        };

        localStorage.setItem('access_token', 'my-token'); // authenticated user

        reachSystemConfiguration(() => {
            cy.intercept('POST', api.auth.tokenRefresh, { body: {} }).as('tokenRefresh');

            interactAndWaitForAuthStatus(() => {
                visitAndWaitForAuthProviders(pagePath);
            }, staticResponseForAuthStatusOK);

            cy.wait('@tokenRefresh');
        });
    });
});
