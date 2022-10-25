import * as api from '../constants/apiEndpoints';
import { url as loginUrl } from '../constants/LoginPage';
import { selectors as navSelectors } from '../constants/TopNavigation';
import withAuth from '../helpers/basicAuth';
import { visitMainDashboard } from '../helpers/main';
import { interactAndWaitForResponses } from '../helpers/request';

const requestConfigForLogout = {
    routeMatcherMap: {
        logout: {
            method: 'POST',
            url: api.auth.logout,
        },
    },
};

const staticResponseMapForLogout = {
    logout: {
        body: {},
    },
};

describe('Logout', () => {
    withAuth();

    it('go to login page after logout on user menu', () => {
        visitMainDashboard();

        interactAndWaitForResponses(
            () => {
                cy.get(navSelectors.menuButton).click();
                cy.get(navSelectors.menuList.logoutButton).click();
            },
            requestConfigForLogout,
            staticResponseMapForLogout
        );

        cy.location('pathname').should('eq', loginUrl);
    });
});
