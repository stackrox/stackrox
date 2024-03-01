import { visit } from '../helpers/visit';
import { interactAndWaitForResponses } from '../helpers/request';

const loginURL = '/login';

const dashboardURL = '/main/dashboard';

const username = Cypress.env('ROX_USERNAME');
const password = Cypress.env('ROX_PASSWORD');

const loginAlias = 'login';
const routeMatcherMapForLogin = {
    [loginAlias]: {
        method: 'GET',
        url: '/v1/login/authproviders',
    },
};

const routeMatcherMapForSubmit = {
    [loginAlias]: {
        method: 'POST',
        url: '/v1/authProviders/exchangeToken',
    },
};

describe('Login', () => {
    it('go to dashboard after login', () => {
        visit(loginURL, routeMatcherMapForLogin);

        interactAndWaitForResponses(() => {
            cy.get('input[name=username]').type(username);
            cy.get('input[name=password]').type(password);
            cy.get('button[type=submit]').click();
        }, routeMatcherMapForSubmit);
        cy.location('pathname').should('eq', dashboardURL);

        cy.getAllLocalStorage().then((result) => {
            expect(result).to.be.empty;
        });

        cy.getCookie('RoxAccessToken').should('exist');
    });
});
