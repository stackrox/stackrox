import { authProvidersUrl, selectors } from '../../constants/AccessControlPage';
import { permissions as permissionsApi } from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';

// TODO Fix 'v1/authProviders*' without initial slash and with asterisk in apiEndpoints?
const authProvidersList = '/v1/authProviders';

describe('Access Control Auth providers', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_SCOPED_ACCESS_CONTROL_V2')) {
            this.skip();
        }
    });

    function visitAuthProviders() {
        cy.intercept('GET', authProvidersList).as('GetAuthProviders');
        cy.visit(authProvidersUrl);
        cy.wait('@GetAuthProviders');
    }

    it('displays alert if no permission for AuthProvider', () => {
        cy.intercept('GET', permissionsApi.mypermissions, {
            fixture: 'auth/mypermissionsMinimalAccess.json',
        }).as('GetMyPermissions');
        cy.visit(authProvidersUrl);
        cy.wait('@GetMyPermissions');

        cy.get(selectors.h1).should('have.text', 'Access Control');
        cy.get(selectors.navLink).should('not.exist');
        cy.get(selectors.h2).should('not.exist');
        cy.get(selectors.alertTitle).should(
            'contain', // instead of have.text because of "Info alert:" for screen reader
            'You do not have permission to view Access Control'
        );
    });

    it('list has headings, link, button, and table head cells', () => {
        visitAuthProviders();

        cy.get(selectors.h1).should('have.text', 'Access Control');
        cy.get(selectors.navLinkCurrent).should('have.text', 'Auth providers');
        cy.get(selectors.h2).should('have.text', 'Auth Providers'); // TODO sentence case
        cy.get(selectors.list.authProviders.addButton).should('have.text', 'Add auth provider'); // TODO Create?

        cy.get(`${selectors.list.th}:contains("Name")`);
        cy.get(`${selectors.list.th}:contains("Type")`);
        cy.get(`${selectors.list.th}:contains("Minimum access role")`);
        cy.get(`${selectors.list.th}:contains("Rules")`); // TODO Auth providers?
    });
});
