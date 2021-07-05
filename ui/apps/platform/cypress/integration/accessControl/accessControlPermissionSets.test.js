import { permissionSetsUrl, selectors } from '../../constants/AccessControlPage';
import {
    permissions as permissionsApi,
    permissionSets as permissionSetsApi,
} from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';

describe('Access Control Permission sets', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_SCOPED_ACCESS_CONTROL_V2')) {
            this.skip();
        }
    });

    function visitPermissionSets() {
        cy.intercept('GET', permissionSetsApi.list).as('GetPermissionSets');
        cy.visit(permissionSetsUrl);
        cy.wait('@GetPermissionSets');
    }

    it('displays alert if no permission for AuthProvider', () => {
        cy.intercept('GET', permissionsApi.mypermissions, {
            fixture: 'auth/mypermissionsMinimalAccess.json',
        }).as('GetMyPermissions');
        cy.visit(permissionSetsUrl);
        cy.wait('@GetMyPermissions');

        cy.get(selectors.h1).should('have.text', 'Access Control');
        cy.get(selectors.navLink).should('not.exist');
        cy.get(selectors.h2).should('not.exist');
        cy.get(selectors.alertTitle).should(
            'contain', // not have.text because it contains "Info alert:" for screen reader
            'You do not have permission to view Access Control'
        );
    });

    it('list has headings, link, button, and table head cells', () => {
        visitPermissionSets();

        cy.get(selectors.h1).should('have.text', 'Access Control');
        cy.get(selectors.navLinkCurrent).should('have.text', 'Permission sets');
        cy.get(selectors.h2).should('have.text', 'Permission sets');
        cy.get(selectors.list.createButton).should('have.text', 'Create permission set');

        cy.get(`${selectors.list.th}:contains("Name")`);
        cy.get(`${selectors.list.th}:contains("Description")`);
        cy.get(`${selectors.list.th}:contains("Roles")`);
    });

    it('list has default permission sets', () => {
        visitPermissionSets();

        cy.get(`${selectors.list.tdLinkName}:contains("Admin")`);
        cy.get(`${selectors.list.tdLinkName}:contains("Analyst")`);
        cy.get(`${selectors.list.tdLinkName}:contains("Continuous Integration")`);
        cy.get(`${selectors.list.tdLinkName}:contains("None")`);
        cy.get(`${selectors.list.tdLinkName}:contains("Sensor Creator")`);
    });

    it('list link goes to form which has headings, link, and label instead of button', () => {
        visitPermissionSets();

        cy.get(`${selectors.list.tdLinkName}:contains("Admin")`).click();

        cy.get(selectors.h1).should('have.text', 'Access Control');
        cy.get(selectors.navLinkCurrent).should('have.text', 'Permission sets');
        cy.get(selectors.h2).should('have.text', 'Admin');
        cy.get(selectors.form.notEditableLabel).should('exist');
        cy.get(selectors.form.editButton).should('not.exist');
    });

    it('list link goes to form which has disabled input values', () => {
        visitPermissionSets();

        cy.get(`${selectors.list.tdLinkName}:contains("Admin")`).click();

        cy.get(selectors.form.inputName).should('be.disabled').should('have.value', 'Admin');
        cy.get(selectors.form.inputDescription).should('be.disabled');
    });
});
