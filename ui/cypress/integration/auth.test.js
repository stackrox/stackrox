import { url as loginUrl, selectors } from './pages/LoginPage';
import { url as complianceUrl } from './pages/CompliancePage';
import * as api from './apiEndpoints';

describe('Authentication', () => {
    const setupAuth = (landingUrl, authStatusValid = true) => {
        cy.server();
        cy.fixture('auth/authProviders.json').as('authProviders');
        cy.route('GET', api.auth.authProviders, '@authProviders').as('authProviders');

        if (authStatusValid) {
            cy.route('GET', api.auth.authStatus, {}).as('authStatus');
        } else {
            cy
                .route({
                    method: 'GET',
                    url: api.auth.authStatus,
                    status: 401,
                    response: {}
                })
                .as('authStatus');
        }

        cy.visit(landingUrl);
        cy.wait('@authProviders');
    };

    it('should redirect user to login page, authenticate and redirect to the requested page', () => {
        setupAuth(complianceUrl);
        cy.url().should('contain', loginUrl);
        cy.get(selectors.providerSelect).should('have.text', 'auth-provider-name');
        cy.get(selectors.loginButton).click(); // stubbed auth provider will simulate OIDC redirect with 'my-token'
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
});
