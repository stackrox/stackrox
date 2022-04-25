import addSeconds from 'date-fns/add_seconds';

import { url as loginUrl, selectors } from '../constants/LoginPage';
import { selectors as navSelectors } from '../constants/TopNavigation';
import { url as dashboardURL } from '../constants/DashboardPage';

import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth'; // used to make logout test less flakey

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
        cy.intercept('GET', api.clusters.list, { fixture: 'clusters/health.json' }).as('clusters');
        cy.intercept('GET', api.search.options, { fixture: 'search/metadataOptions.json' }).as(
            'searchOptions'
        );
        cy.intercept('GET', api.alerts.countsByCluster, { body: {} }).as('countsByCluster');
        cy.intercept('GET', api.alerts.countsByCategory, { body: {} }).as('countsByCategory');
        cy.intercept('GET', api.dashboard.timeseries, { body: {} }).as('alertsByTimeseries');
        cy.intercept('GET', api.risks.riskyDeployments, { body: {} }).as('deployments');
        cy.intercept('POST', api.logs, { body: {} }).as('logs');
    };

    it('should redirect user to login page, authenticate and redirect to the requested page', () => {
        stubAPIs();
        setupAuth(dashboardURL, AUTHENTICATED);
        cy.location('pathname').should('eq', loginUrl);
        cy.get(selectors.providerSelect).should('have.text', 'auth-provider-name');
        cy.get(selectors.loginButton).click(); // stubbed auth provider will simulate redirect with 'my-token'
        // Replace Authorization for route method with authorization for intercept method.
        cy.wait('@authStatus').its('request.headers.authorization').should('eq', 'Bearer my-token');
        cy.location('pathname').should('eq', dashboardURL);
    });

    it('should allow authenticated user to enter', () => {
        stubAPIs();
        localStorage.setItem('access_token', 'my-token'); // simulate authenticated user
        setupAuth(dashboardURL, AUTHENTICATED);

        cy.wait('@authStatus');

        cy.location('pathname').should('eq', dashboardURL);
    });

    it('should logout previously authenticated user with invalid token', () => {
        stubAPIs();
        localStorage.setItem('access_token', 'my-token'); // invalid token
        setupAuth(dashboardURL, UNAUTHENTICATED);

        cy.wait('@authStatus');

        cy.location('pathname').should('eq', loginUrl);
    });

    // TODO: Fix it, see ROX-4983 for more explanation
    it.skip('should request token refresh 30 sec in advance', () => {
        stubAPIs();
        cy.intercept('POST', api.auth.tokenRefresh, { body: {} }).as('tokenRefresh');
        localStorage.setItem('access_token', 'my-token'); // authenticated user

        const expiryDate = addSeconds(Date.now(), 33); // +3 sec should be enough
        setupAuth(dashboardURL, AUTHENTICATED, {
            expires: expiryDate.toISOString(),
        });

        cy.wait('@tokenRefresh');
    });

    // the logout test has its own describe block, which uses our withAuth() helper function
    //   to log in with a real auth token
    //   because after a Cypress upgrade, using a fake token on this test became flakey
    describe('Logout', () => {
        withAuth();

        // turning off for now, because of an issue with Cypress
        // see https://srox.slack.com/archives/C7ERNFL0M/p1596839383218700
        it.skip('should logout user by request', () => {
            cy.intercept('POST', api.auth.logout, { body: {} }).as('logout');

            cy.visit(dashboardURL);

            cy.get(navSelectors.menuButton).click();
            cy.get(navSelectors.menuList.logoutButton).click();
            cy.wait('@logout');
            cy.location('pathname').should('eq', loginUrl);
        });
    });
});
