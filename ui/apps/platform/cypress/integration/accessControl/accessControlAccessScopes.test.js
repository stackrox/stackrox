import { accessScopesUrl, selectors } from '../../constants/AccessControlPage';
import {
    accessScopes as accessScopesApi,
    permissions as permissionsApi,
} from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';

describe('Access Control Access scopes', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_SCOPED_ACCESS_CONTROL_V2')) {
            this.skip();
        }
    });

    function visitAccessScopes() {
        cy.intercept('GET', accessScopesApi.list).as('GetAccessScopes');
        cy.visit(accessScopesUrl);
        cy.wait('@GetAccessScopes');
    }

    it('displays alert if no permission for AuthProvider', () => {
        cy.intercept('GET', permissionsApi.mypermissions, {
            fixture: 'auth/mypermissionsMinimalAccess.json',
        }).as('GetMyPermissions');
        cy.visit(accessScopesUrl);
        cy.wait('@GetMyPermissions');

        cy.get(selectors.h1).should('have.text', 'Access Control');
        cy.get(selectors.navLink).should('not.exist');
        cy.get(selectors.h2).should('not.exist');
        cy.get(selectors.alertTitle).should(
            'contain', // not have.text because it contains "Info alert:" for screen reader
            'You do not have permission to view Access Control'
        );
    });

    it('list has headings, link, and button', () => {
        visitAccessScopes();

        cy.get(selectors.h1).should('have.text', 'Access Control');
        cy.get(selectors.navLinkCurrent).should('have.text', 'Access scopes');
        cy.get(selectors.h2).should('have.text', 'Access scopes');
        cy.get(selectors.list.addButton).should('have.text', 'Add access scope');

        // Although no default access scopes, do not assume whether or not table exists.
    });
});
