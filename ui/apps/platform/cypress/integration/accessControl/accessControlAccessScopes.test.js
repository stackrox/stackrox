import { accessScopesUrl, selectors } from '../../constants/AccessControlPage';

import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    accessScopesKey as entitiesKey,
    clickEntityNameInTable,
    visitAccessControlEntities,
    visitAccessControlEntitiesWithStaticResponseForPermissions,
    visitAccessControlEntity,
} from './accessControl.helpers';

const h2 = 'Access scopes';

const defaultNames = ['Unrestricted', 'Deny All'];

describe('Access Control Access scopes', () => {
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
            'You do not have permission to view access scopes.'
        );
    });

    it('list has heading, button, and table head cells', () => {
        visitAccessControlEntities(entitiesKey);

        // Table has plural noun in title.
        cy.title().should('match', getRegExpForTitleWithBranding(`Access Control - ${h2}`));

        cy.contains('h2', /^\d+ results? found$/);
        cy.get(selectors.list.createButton).should('have.text', 'Create access scope');

        cy.get('th:contains("Name")');
        cy.get('th:contains("Description")');
        cy.get('th:contains("Roles")');
    });

    it('list has default names', () => {
        visitAccessControlEntities(entitiesKey);

        defaultNames.forEach((name) => {
            cy.get(`td[data-label="Name"] a:contains("${name}")`);
        });
    });

    it('list link for default Deny All goes to form which has label instead of button and disabled input values', () => {
        visitAccessControlEntities(entitiesKey);

        const name = 'Deny All';
        clickEntityNameInTable(entitiesKey, name);

        // Form has singular noun in title.
        cy.title().should('match', getRegExpForTitleWithBranding(`Access Control - Access scope`));

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("${name}")`);

        cy.get('h1').should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');

        cy.get('h2').should('have.text', name);
        cy.get(selectors.form.notEditableLabel).should('exist');
        cy.get(selectors.form.editButton).should('not.exist');

        cy.get(selectors.form.inputName).should('be.disabled');
        cy.get(selectors.form.inputDescription).should('be.disabled');
    });

    it('displays message instead of form if entity id does not exist', () => {
        const entityId = 'bogus';

        visitAccessControlEntity(entitiesKey, entityId);

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2)`).should('not.exist');

        cy.get('h1').should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');
        cy.get('h2').should('not.exist');

        cy.get(selectors.notFound.title).should('have.text', 'Access scope does not exist');
        cy.get(selectors.notFound.a)
            .should('have.text', h2)
            .should('have.attr', 'href', accessScopesUrl);
    });
});
