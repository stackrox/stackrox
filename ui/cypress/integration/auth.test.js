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
        cy.server();
        cy.route('GET', api.auth.loginAuthProviders, 'fixture:auth/authProviders.json').as(
            'authProviders'
        );
        cy.route({
            method: 'GET',
            url: api.auth.authStatus,
            status: authStatusValid ? 200 : 401,
            response: authStatusResponse,
        }).as('authStatus');

        cy.visit(landingUrl);
        cy.wait('@authProviders');
    };

    const stubAPIs = () => {
        cy.server();
        // Cypress routes have an override behaviour, so defining this first makes it the fallback.
        cy.route(/.*/, {}).as('everythingElse');
        cy.route('GET', api.clusters.list, 'fixture:clusters/couple.json').as('clusters');
        cy.route('GET', api.search.options, 'fixture:search/metadataOptions.json').as(
            'searchOptions'
        );
        cy.route('GET', api.alerts.countsByCluster, {}).as('countsByCluster');
        cy.route('GET', api.alerts.countsByCategory, {}).as('countsByCategory');
        cy.route('GET', api.dashboard.timeseries, {}).as('alertsByTimeseries');
        cy.route('GET', api.risks.riskyDeployments, {}).as('deployments');
        cy.route('GET', api.licenses.list, { response: { licenses: [{ status: 'VALIDs' }] } }).as(
            'licenses'
        );
        cy.route('POST', api.logs, {}).as('logs');
    };

    it('should redirect user to login page, authenticate and redirect to the requested page', () => {
        stubAPIs();
        setupAuth(dashboardURL, AUTHENTICATED);
        cy.server();
        cy.route('GET', api.clusters.list, 'fixture:clusters/couple.json').as('clusters');
        cy.route('GET', api.search.options, 'fixture:search/metadataOptions.json').as(
            'searchOptions'
        );
        cy.url().should('contain', loginUrl);
        cy.get(selectors.providerSelect).should('have.text', 'auth-provider-name');
        cy.get(selectors.loginButton).click(); // stubbed auth provider will simulate redirect with 'my-token'
        cy.wait('@authStatus').then((xhr) => {
            expect(xhr.request.headers.Authorization).to.eq('Bearer my-token');
        });
        cy.url().should('contain', dashboardURL);
    });

    it('should allow authenticated user to enter', () => {
        localStorage.setItem('access_token', 'my-token'); // simulate authenticated user
        stubAPIs();
        setupAuth(dashboardURL, AUTHENTICATED);

        cy.wait('@authStatus');

        cy.url().should('contain', dashboardURL);
    });

    it('should logout previously authenticated user with invalid token', () => {
        localStorage.setItem('access_token', 'my-token'); // invalid token
        stubAPIs();
        setupAuth(dashboardURL, UNAUTHENTICATED);

        cy.wait('@authStatus');

        cy.url().should('contain', loginUrl);
    });

    it('should request token refresh 30 sec in advance', () => {
        localStorage.setItem('access_token', 'my-token'); // authenticated user
        stubAPIs();
        cy.route('POST', api.auth.tokenRefresh, {}).as('tokenRefresh');

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

        it('should logout user by request', () => {
            cy.server();
            cy.route('POST', api.auth.logout, {}).as('logout');

            cy.visit(dashboardURL);

            cy.get(navSelectors.menuButton).click();
            cy.get(navSelectors.logoutButton).click();
            cy.wait('@logout');
            cy.url().should('contain', loginUrl);
        });
    });
});
