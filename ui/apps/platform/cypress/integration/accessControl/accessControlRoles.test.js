import { rolesUrl, selectors } from '../../constants/AccessControlPage';

import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    clickEntityNameInTable,
    rolesKey as entitiesKey,
    visitAccessControlEntities,
    visitAccessControlEntitiesWithStaticResponseForPermissions,
    visitAccessControlEntity,
} from './accessControl.helpers';

// Migration from cy.server and cy.route to cy.intercept fails for /v1/roles/* imported from apiEndpoints.
const rolesApi = {
    list: '/v1/roles',
};

const h2 = 'Roles';

const defaultNames = ['Admin', 'Analyst', 'Continuous Integration', 'None', 'Sensor Creator'];

describe('Access Control Roles', () => {
    withAuth();

    it('displays alert if no permission', () => {
        const staticResponseForPermissions = {
            fixture: 'auth/mypermissionsMinimalAccess.json',
        };
        visitAccessControlEntitiesWithStaticResponseForPermissions(
            entitiesKey,
            staticResponseForPermissions
        );

        cy.get(selectors.alertTitle).should(
            'contain', // not have.text because it contains "Info alert:" for screen reader
            'You do not have permission to view roles.'
        );
    });

    it('list has heading, button, and table head cells', () => {
        visitAccessControlEntities(entitiesKey);

        // Table has plural noun in title.
        cy.title().should('match', getRegExpForTitleWithBranding(`Access Control - Roles`));

        cy.get(selectors.breadcrumbNav).should('not.exist');

        cy.contains('h2', /^\d+ results? found$/);
        cy.get(selectors.list.createButton).should('have.text', 'Create role');

        cy.get('th:contains("Name")');
        cy.get('th:contains("Description")');
        cy.get('th:contains("Permission set")');
        cy.get('th:contains("Access scope")');
    });

    it('list has default names', () => {
        visitAccessControlEntities(entitiesKey);

        const { tdPermissionSetLink, tdAccessScope } = selectors.list.roles;

        cy.get(selectors.list.tdNameLink).then(($tds) => {
            $tds.get().forEach((td, index) => {
                const roleName = td.textContent;
                if (defaultNames.includes(roleName)) {
                    cy.get(`${tdPermissionSetLink}:eq(${index})`).should('have.text', roleName);
                    cy.get(`${tdAccessScope}:eq(${index})`).should('have.text', 'Unrestricted');
                }
            });
        });
    });

    it('list link goes to form which has label instead of button and disabled input values', () => {
        visitAccessControlEntities(entitiesKey);

        const name = 'Admin';
        clickEntityNameInTable(entitiesKey, name);

        // Form has singular noun in title.
        cy.title().should('match', getRegExpForTitleWithBranding(`Access Control - Role`));

        cy.get('h1').should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');

        cy.get('h2').should('have.text', name);
        cy.get(selectors.form.notEditableLabel).should('exist');
        cy.get(selectors.form.editButton).should('not.exist');

        cy.get(selectors.form.inputName).should('be.disabled');
        cy.get(selectors.form.inputDescription).should('be.disabled');

        const { getRadioPermissionSetForName, getRadioAccessScopeForName } = selectors.form.role;

        defaultNames.forEach((defaultName) => {
            cy.get(getRadioPermissionSetForName(defaultName)).should('be.disabled');
        });

        cy.get(getRadioAccessScopeForName('Unrestricted')).should('be.disabled');
    });

    describe('direct link to default role', () => {
        const { getRadioPermissionSetForName, getRadioAccessScopeForName } = selectors.form.role;

        defaultNames.forEach((roleName) => {
            it(`${roleName} has corresponding permission set and no access scope`, () => {
                visitAccessControlEntity(entitiesKey, roleName);

                cy.get(selectors.form.inputName).should('have.value', roleName);

                defaultNames.forEach((defaultName) => {
                    cy.get(getRadioPermissionSetForName(defaultName)).should(
                        defaultName === roleName ? 'be.checked' : 'not.be.checked'
                    );
                });

                cy.get(getRadioAccessScopeForName('Unrestricted')).should('be.checked');
            });
        });
    });

    it('adds a new role and form disables name input when editing an existing role', () => {
        visitAccessControlEntities(entitiesKey);

        cy.get(selectors.list.createButton).click();

        cy.get('h2').should('have.text', 'Create role');
        cy.get(selectors.form.notEditableLabel).should('not.exist');
        cy.get(selectors.form.editButton).should('not.exist');
        cy.get(selectors.form.saveButton).should('be.disabled');
        cy.get(selectors.form.cancelButton).should('be.enabled');

        cy.get(selectors.form.inputName).should('be.enabled').should('have.value', '');
        cy.get(selectors.form.inputDescription).should('be.enabled').should('have.value', '');

        const name = `Role-${new Date().toISOString()}`;
        const description =
            'adds a new role and form disables name input when editing an existing role';
        const permissionSetName = 'None';
        const accessScopeName = 'Unrestricted';

        cy.get(selectors.form.inputName).type(name);
        cy.get(selectors.form.inputDescription).type(description);
        cy.get(selectors.form.role.getRadioPermissionSetForName(permissionSetName)).click();
        cy.get(selectors.form.role.getRadioAccessScopeForName(accessScopeName))
            .should('be.enabled')
            .should('be.checked');

        cy.intercept('POST', `${rolesApi.list}/${name}`).as('PostRoles');
        cy.get(selectors.form.saveButton).click();
        cy.wait('@PostRoles');

        cy.contains('h2', /^\d+ results? found$/).should('exist');
        cy.get(`${selectors.list.tdNameLink}:contains("${name}")`).click();

        cy.get('h2').should('have.text', name);
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

        // TODO go back to list and delete role to clean up after the test
    });

    it('displays message instead of form if entity id does not exist', () => {
        const entityId = 'bogus';

        visitAccessControlEntity(entitiesKey, entityId);

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2)`).should('not.exist');

        cy.get('h1').should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');
        cy.get('h2').should('not.exist');

        cy.get(selectors.notFound.title).should('have.text', 'Role does not exist');
        cy.get(selectors.notFound.a).should('have.text', h2).should('have.attr', 'href', rolesUrl);
    });
});
