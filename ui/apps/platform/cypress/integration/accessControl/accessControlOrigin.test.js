import withAuth from '../../helpers/basicAuth';

import {
    accessScopesAlias,
    accessScopesKey,
    authProvidersAlias,
    authProvidersKey,
    clickEntityNameInTable,
    groupsAlias,
    permissionSetsAlias,
    permissionSetsKey,
    rolesAlias,
    rolesKey,
    visitAccessControlEntities,
} from './accessControl.helpers';
import { selectors } from './accessControl.selectors';

describe('Access Control declarative resources', () => {
    withAuth();

    // Access scopes.
    it('declarative access scopes in list view have correct origin and are disabled', () => {
        const staticResponseMap = {
            [accessScopesAlias]: {
                fixture: 'auth/declarativeAccessScope.json',
            },
        };
        visitAccessControlEntities(accessScopesKey, staticResponseMap);

        cy.get('td[data-label="Origin"]').should('have.text', 'Declarative');
        cy.get('td.pf-c-table__action button[aria-label="Actions"]').should('be.disabled');
    });

    it('list link for declarative access scope goes to form which has label instead of button and disabled input values', () => {
        const staticResponseMap = {
            [accessScopesAlias]: {
                fixture: 'auth/declarativeAccessScope.json',
            },
        };
        visitAccessControlEntities(accessScopesKey, staticResponseMap);

        const entityName = 'access-scope-test-name';
        clickEntityNameInTable(accessScopesKey, entityName);

        cy.get(`h2:contains("${entityName}")`);
        cy.get(`li.pf-c-breadcrumb__item:nth-child(2):contains("${entityName}")`);

        cy.get('.pf-c-label:contains("Declarative")').should('exist');

        cy.get(selectors.form.notEditableLabel).should('exist');
        cy.get(selectors.form.editButton).should('not.exist');

        cy.get(selectors.form.inputName).should('be.disabled');
        cy.get(selectors.form.inputDescription).should('be.disabled');
    });

    // Auth providers.
    it('declarative auth providers in list view have correct origin and are disabled', () => {
        const staticResponseMap = {
            [authProvidersAlias]: {
                fixture: 'auth/declarativeAuthProvider.json',
            },
            [groupsAlias]: {
                fixture: 'auth/declarativeGroups.json',
            },
        };
        visitAccessControlEntities(authProvidersKey, staticResponseMap);

        cy.get('td[data-label="Origin"]').should('have.text', 'Declarative');
        cy.get(`td.pf-c-table__action .pf-c-dropdown__toggle`).click();
        cy.get(
            `td.pf-c-table__action button[role="menuitem"]:contains("Delete auth provider")`
        ).should('have.attr', 'aria-disabled', 'true');
    });

    it('list link for declarative auth provider + groups goes to form which has label instead of button and disabled input values', () => {
        const staticResponseMap = {
            [authProvidersAlias]: {
                fixture: 'auth/declarativeAuthProvider.json',
            },
            [groupsAlias]: {
                fixture: 'auth/declarativeGroups.json',
            },
        };
        visitAccessControlEntities(authProvidersKey, staticResponseMap);

        const entityName = 'auth-provider-1';
        clickEntityNameInTable(authProvidersKey, entityName);

        cy.get(`h2:contains("${entityName}")`);
        cy.get(`li.pf-c-breadcrumb__item:nth-child(2):contains("${entityName}")`);

        cy.get('.pf-c-label:contains("Declarative")').should('exist');

        cy.get('button:contains("Edit minimum role and rules")').should('be.enabled');

        cy.get(selectors.form.authProvider.saml.inputMetadataURL).should('be.disabled');
        cy.get(selectors.form.authProvider.saml.inputServiceProviderIssuer).should('be.disabled');
        cy.get(selectors.form.authProvider.saml.selectConfiguration).should('be.disabled');

        cy.get('input[id="groups[0].props.value"]').should('be.disabled');
        cy.get('input[id="groups[0].props.key-select-typeahead"]').should('be.disabled');
        cy.get('button[id="groups[0].roleName"]').should('be.disabled');
        cy.get('button[aria-label="Delete rule"]').should('not.exist');
    });

    // Permission sets.
    it('declarative permission sets in list view have correct origin and are disabled', () => {
        const staticResponseMap = {
            [permissionSetsAlias]: {
                fixture: 'auth/declarativePermissionSet.json',
            },
        };
        visitAccessControlEntities(permissionSetsKey, staticResponseMap);

        cy.get('td[data-label="Origin"]').should('have.text', 'Declarative');
        cy.get('td.pf-c-table__action button[aria-label="Actions"]').should('be.disabled');
    });

    it('list link for declarative permission set goes to form which has label instead of button and disabled input values', () => {
        const staticResponseMap = {
            [permissionSetsAlias]: {
                fixture: 'auth/declarativePermissionSet.json',
            },
        };
        visitAccessControlEntities(permissionSetsKey, staticResponseMap);

        const entityName = 'permission-set-test-name';
        clickEntityNameInTable(permissionSetsKey, entityName);

        cy.get(`h2:contains("${entityName}")`);
        cy.get(`li.pf-c-breadcrumb__item:nth-child(2):contains("${entityName}")`);

        cy.get('.pf-c-label:contains("Declarative")').should('exist');

        cy.get(selectors.form.notEditableLabel).should('exist');
        cy.get(selectors.form.editButton).should('not.exist');

        cy.get(selectors.form.inputName).should('be.disabled');
        cy.get(selectors.form.inputDescription).should('be.disabled');
    });

    // Roles.
    it('declarative roles in list view have correct origin and are disabled', () => {
        const staticResponseMap = {
            [rolesAlias]: {
                fixture: 'auth/declarativeRole.json',
            },
        };
        visitAccessControlEntities(rolesKey, staticResponseMap);

        cy.get('td[data-label="Origin"]').should('have.text', 'Declarative');
        cy.get('td.pf-c-table__action button[aria-label="Actions"]').should('be.disabled');
    });

    it('list link for declarative role goes to form which has label instead of button and disabled input values', () => {
        const staticResponseMap = {
            [rolesAlias]: {
                fixture: 'auth/declarativeRole.json',
            },
        };
        visitAccessControlEntities(rolesKey, staticResponseMap);

        const entityName = 'test-role-name';
        clickEntityNameInTable(rolesKey, entityName);

        cy.get(`h2:contains("${entityName}")`);
        cy.get(`li.pf-c-breadcrumb__item:nth-child(2):contains("${entityName}")`);

        cy.get('.pf-c-label:contains("Declarative")').should('exist');

        cy.get(selectors.form.notEditableLabel).should('exist');
        cy.get(selectors.form.editButton).should('not.exist');

        cy.get(selectors.form.inputName).should('be.disabled');
        cy.get(selectors.form.inputDescription).should('be.disabled');
    });
});
