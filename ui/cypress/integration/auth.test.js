import { url as loginUrl, selectors } from './constants/LoginPage';
import { url as dashboardURL } from './constants/DashboardPage';

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

    const stubAPIs = () => {
        cy.server();
        // Cypress routes have an override behaviour, so defining this first makes it the fallback.
        cy.route(/.*/, {}).as('everythingElse');
        cy.route('GET', api.clusters.list, 'fixture:clusters/couple.json').as('clusters');
        cy.route('GET', api.search.options, 'fixture:search/metadataOptions.json').as(
            'searchOptions'
        );
        cy.route('GET', api.summary.counts, {}).as('summaryCounts');
        cy.route('GET', api.alerts.countsByCluster, {}).as('countsByCluster');
        cy.route('GET', api.alerts.countsByCategory, {}).as('countsByCategory');
        cy.route('GET', api.dashboard.timeseries, {}).as('alertsByTimeseries');
        cy.route('GET', api.risks.riskyDeployments, {}).as('deployments');
        cy.route('GET', api.licenses.list, { response: { licenses: [{ status: 'VALIDs' }] } }).as(
            'licenses'
        );
        cy.route('POST', api.logs, {}).as('logs');
    };

    xit('should redirect user to login page, authenticate and redirect to the requested page', () => {
        stubAPIs();
        setupAuth(dashboardURL);
        cy.server();
        cy.route('GET', api.clusters.list, 'fixture:clusters/couple.json').as('clusters');
        cy.route('GET', api.search.options, 'fixture:search/metadataOptions.json').as(
            'searchOptions'
        );
        cy.url().should('contain', loginUrl);
        cy.get(selectors.providerSelect).should('have.text', 'auth-provider-name');
        cy.get(selectors.loginButton).click(); // stubbed auth provider will simulate redirect with 'my-token'
        cy.wait('@authStatus').then(xhr => {
            expect(xhr.request.headers.Authorization).to.eq('Bearer my-token');
        });
        cy.url().should('contain', dashboardURL);
    });

    xit('should allow authenticated user to enter', () => {
        localStorage.setItem('access_token', 'my-token'); // simulate authenticated user
        stubAPIs();
        setupAuth(dashboardURL);
        cy.url().should('contain', dashboardURL);
    });

    it('should logout previously authenticated user with invalid token', () => {
        localStorage.setItem('access_token', 'my-token'); // invalid token
        stubAPIs();
        setupAuth(dashboardURL, false);
        cy.url().should('contain', loginUrl);
    });

    // TODO(ROX-990): Fix and re-enable this test. It was flaky on OpenShift and K8s (failure rate was higher on OpenShift though).
    xit('should logout user by request', () => {
        localStorage.setItem('access_token', 'my-token'); // authenticated user
        setupAuth(dashboardURL);
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
        stubAPIs();
        cy.route({
            method: 'GET',
            url: api.summary.counts,
            status: 401,
            delay: 200,
            response: {},
            onRequest: () => {
                localStorage.setItem('access_token', 'new-token');
            }
        }).as('summaryCountsTokenChanging');
        setupAuth(dashboardURL);

        cy.wait('@summaryCountsTokenChanging').then(xhr => {
            expect(xhr.request.headers.Authorization).to.eq('Bearer first-token');
        });
        // should retry request with a new token
        cy.wait('@summaryCountsTokenChanging').then(xhr => {
            expect(xhr.request.headers.Authorization).to.eq('Bearer new-token');
        });
    });
});
