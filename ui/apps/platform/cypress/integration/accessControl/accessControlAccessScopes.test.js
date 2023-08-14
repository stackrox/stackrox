import withAuth from '../../helpers/basicAuth';
import { assertCannotFindThePage } from '../../helpers/visit';

import {
    accessScopesKey as entitiesKey,
    assertAccessControlEntityDoesNotExist,
    clickEntityNameInTable,
    visitAccessControlEntities,
    visitAccessControlEntitiesWithStaticResponseForPermissions,
    visitAccessControlEntity,
} from './accessControl.helpers';
import { selectors } from './accessControl.selectors';

const defaultNames = ['Unrestricted', 'Deny All'];

describe('Access Control Access scopes', () => {
    withAuth();

    it('cannot find the page if no permission', () => {
        const staticResponseForPermissions = {
            fixture: 'auth/mypermissionsMinimalAccess.json',
        };
        visitAccessControlEntitiesWithStaticResponseForPermissions(
            entitiesKey,
            staticResponseForPermissions
        );

        assertCannotFindThePage();
    });

    it('list has heading, button, and table head cells', () => {
        visitAccessControlEntities(entitiesKey);

        cy.contains('h2', /^\d+ results? found$/);

        cy.get('button:contains("Create access scope")');

        cy.get('th:contains("Name")');
        cy.get('th:contains("Origin")');
        cy.get('th:contains("Description")');
        cy.get('th:contains("Roles")');
        cy.get('th[aria-label="Row actions"]');
    });

    it('list has default names', () => {
        visitAccessControlEntities(entitiesKey);

        defaultNames.forEach((defaultName) => {
            cy.get(`td[data-label="Name"] a:contains("${defaultName}")`);
        });
    });

    it('list link for default Deny All goes to form which has label instead of button and disabled input values', () => {
        visitAccessControlEntities(entitiesKey);

        const entityName = 'Deny All';
        clickEntityNameInTable(entitiesKey, entityName);

        cy.get(`h2:contains("${entityName}")`);
        cy.get(`li.pf-c-breadcrumb__item:nth-child(2):contains("${entityName}")`);

        cy.get(selectors.form.notEditableLabel).should('exist');
        cy.get(selectors.form.editButton).should('not.exist');

        cy.get(selectors.form.inputName).should('be.disabled');
        cy.get(selectors.form.inputDescription).should('be.disabled');
    });

    it('displays message instead of form if entity id does not exist', () => {
        const entityId = 'bogus';
        visitAccessControlEntity(entitiesKey, entityId);

        assertAccessControlEntityDoesNotExist(entitiesKey);
    });
});
