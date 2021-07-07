import { rolesUrl, selectors } from '../../constants/AccessControlPage';
import { permissions as permissionsApi } from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';

// Migration from cy.server and cy.route to cy.intercept fails for /v1/roles/* imported from apiEndpoints.
const rolesApiList = '/v1/roles'; // instead of rolesApi.list

describe('Access Control Roles', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_SCOPED_ACCESS_CONTROL_V2')) {
            this.skip();
        }
    });

    function visitRoles() {
        cy.intercept('GET', rolesApiList).as('GetRoles');
        cy.visit(rolesUrl);
        cy.wait('@GetRoles');
    }

    it('displays alert if no permission for AuthProvider', () => {
        cy.intercept('GET', permissionsApi.mypermissions, {
            fixture: 'auth/mypermissionsMinimalAccess.json',
        }).as('GetMyPermissions');
        cy.visit(rolesUrl);
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
        visitRoles();

        cy.get(selectors.h1).should('have.text', 'Access Control');
        cy.get(selectors.navLinkCurrent).should('have.text', 'Roles');
        cy.get(selectors.h2).should('have.text', 'Roles');
        cy.get(selectors.list.addButton).should('have.text', 'Add role');

        cy.get(`${selectors.list.th}:contains("Name")`);
        cy.get(`${selectors.list.th}:contains("Description")`);
        cy.get(`${selectors.list.th}:contains("Permission set")`);
        cy.get(`${selectors.list.th}:contains("Access scope")`);
    });

    it('list has default roles', () => {
        visitRoles();

        cy.get(`${selectors.list.tdLinkName}:contains("Admin")`);
        cy.get(`${selectors.list.tdLinkName}:contains("Analyst")`);
        cy.get(`${selectors.list.tdLinkName}:contains("Continuous Integration")`);
        cy.get(`${selectors.list.tdLinkName}:contains("None")`);
        cy.get(`${selectors.list.tdLinkName}:contains("Sensor Creator")`);
    });

    it('list link goes to form which has headings, link, and label instead of button', () => {
        visitRoles();

        cy.get(`${selectors.list.tdLinkName}:contains("Admin")`).click();

        cy.get(selectors.h1).should('have.text', 'Access Control');
        cy.get(selectors.navLinkCurrent).should('have.text', 'Roles');
        cy.get(selectors.h2).should('have.text', 'Admin');
        cy.get(selectors.form.notEditableLabel).should('exist');
        cy.get(selectors.form.editButton).should('not.exist');
    });

    it('list link goes to form which has disabled input values', () => {
        visitRoles();

        cy.get(`${selectors.list.tdLinkName}:contains("Admin")`).click();

        cy.get(selectors.form.inputName).should('be.disabled').should('have.value', 'Admin');
        cy.get(selectors.form.inputDescription).should('be.disabled');

        cy.get(selectors.form.role.getRadioPermissionSetForName('Admin'))
            .should('be.disabled')
            .should('be.checked');
        cy.get(selectors.form.role.getRadioPermissionSetForName('Analyst'))
            .should('be.disabled')
            .should('not.be.checked');
        cy.get(selectors.form.role.getRadioPermissionSetForName('Continuous Integration'))
            .should('be.disabled')
            .should('not.be.checked');
        cy.get(selectors.form.role.getRadioPermissionSetForName('None'))
            .should('be.disabled')
            .should('not.be.checked');
        cy.get(selectors.form.role.getRadioPermissionSetForName('Sensor Creator'))
            .should('be.disabled')
            .should('not.be.checked');

        cy.get(selectors.form.role.getRadioAccessScopeForName('No access scope'))
            .should('be.disabled')
            .should('be.checked');
    });

    it('creates a new role and form disables name input when editing an existing role', () => {
        visitRoles();

        cy.get(selectors.list.addButton).click();

        cy.get(selectors.h2).should('have.text', 'Add role');
        cy.get(selectors.form.notEditableLabel).should('not.exist');
        cy.get(selectors.form.editButton).should('not.exist');
        cy.get(selectors.form.saveButton).should('be.disabled');
        cy.get(selectors.form.cancelButton).should('be.enabled');

        cy.get(selectors.form.inputName).should('be.enabled').should('have.value', '');
        cy.get(selectors.form.inputDescription).should('be.enabled').should('have.value', '');

        const name = `Role-${new Date().getTime()}`;
        const description = 'New description';
        const permissionSetName = 'None';
        const accessScopeName = 'No access scope';

        cy.get(selectors.form.inputName).type(name);
        cy.get(selectors.form.inputDescription).type(description);
        cy.get(selectors.form.role.getRadioPermissionSetForName(permissionSetName)).click();
        cy.get(selectors.form.role.getRadioAccessScopeForName(accessScopeName))
            .should('be.enabled')
            .should('be.checked');

        cy.intercept('POST', rolesApiList).as('PostRoles');
        cy.get(selectors.form.saveButton).click();
        cy.wait('@PostRoles');

        cy.get(selectors.h2).should('have.text', name);
        cy.get(selectors.form.inputName).should('be.disabled').should('have.value', name);
        cy.get(selectors.form.notEditableLabel).should('not.exist');
        cy.get(selectors.form.editButton).should('exist');
        cy.get(selectors.form.saveButton).should('not.exist');
        cy.get(selectors.form.cancelButton).should('not.exist');

        cy.get(selectors.form.editButton).click();
        cy.get(selectors.form.editButton).should('be.disabled');
        cy.get(selectors.form.inputName).should('be.disabled');

        cy.get(selectors.form.cancelButton).click();
        cy.get(selectors.form.editButton).should('be.enabled');
    });
});
