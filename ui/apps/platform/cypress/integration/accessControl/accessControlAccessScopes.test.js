import { accessScopesUrl, selectors } from '../../constants/AccessControlPage';
import {
    accessScopes as accessScopesApi,
    permissions as permissionsApi,
} from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';

const h1 = 'Access Control';
const h2 = 'Access scopes';

const defaultNames = ['Unrestricted', 'Deny All'];

describe('Access Control Access scopes', () => {
    withAuth();

    function visitAccessScopes() {
        cy.intercept('GET', accessScopesApi.list).as('GetAccessScopes');
        cy.visit(accessScopesUrl);
        cy.wait('@GetAccessScopes');
    }

    it('displays alert if no permission', () => {
        cy.intercept('GET', permissionsApi.mypermissions, {
            fixture: 'auth/mypermissionsMinimalAccess.json',
        }).as('GetMyPermissions');
        cy.visit(accessScopesUrl);
        cy.wait('@GetMyPermissions');

        cy.get(selectors.h1).should('have.text', h1);
        cy.get(selectors.navLink).should('not.exist');

        cy.get(selectors.h2).should('not.exist');

        cy.get(selectors.alertTitle).should(
            'contain', // not have.text because it contains "Info alert:" for screen reader
            'You do not have permission to view Access Control'
        );
    });

    it('list has headings, link, button, and table head cells, and no breadcrumbs', () => {
        visitAccessScopes();

        cy.get(selectors.breadcrumbNav).should('not.exist');

        cy.get(selectors.h1).should('have.text', h1);
        cy.get(selectors.navLinkCurrent).should('have.text', h2);

        cy.contains(selectors.h2, /^\d+ results? found$/).should('exist');
        cy.get(selectors.list.createButton).should('have.text', 'Create access scope');

        cy.get(`${selectors.list.th}:contains("Name")`);
        cy.get(`${selectors.list.th}:contains("Description")`);
        cy.get(`${selectors.list.th}:contains("Roles")`);
    });

    it('list has default names', () => {
        visitAccessScopes();

        defaultNames.forEach((name) => {
            cy.get(`${selectors.list.tdNameLink}:contains("${name}")`);
        });
    });

    it('list link for default Deny All goes to form which has label instead of button and disabled input values', () => {
        visitAccessScopes();

        const name = 'Deny All';
        cy.get(`${selectors.list.tdNameLink}:contains("${name}")`).click();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("${name}")`);

        cy.get(selectors.h1).should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');

        cy.get(selectors.h2).should('have.text', name);
        cy.get(selectors.form.notEditableLabel).should('exist');
        cy.get(selectors.form.editButton).should('not.exist');

        cy.get(selectors.form.inputName).should('be.disabled');
        cy.get(selectors.form.inputDescription).should('be.disabled');
    });

    it('displays message instead of form if entity id does not exist', () => {
        cy.intercept('GET', accessScopesApi.list).as('GetAccessScopes');
        cy.visit(`${accessScopesUrl}/bogus`);
        cy.wait('@GetAccessScopes');

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2)`).should('not.exist');

        cy.get(selectors.h1).should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');
        cy.get(selectors.h2).should('not.exist');

        cy.get(selectors.notFound.title).should('have.text', 'Access scope does not exist');
        cy.get(selectors.notFound.a)
            .should('have.text', h2)
            .should('have.attr', 'href', accessScopesUrl);
    });
});
