import { authProvidersUrl, selectors } from '../../constants/AccessControlPage';
import { permissions as permissionsApi } from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';

// TODO Fix 'v1/authProviders*' without initial slash and with asterisk in apiEndpoints?
const authProvidersApi = {
    list: '/v1/authProviders',
};

const groupsApi = {
    list: '/v1/groups',
};

const h1 = 'Access Control';
const h2 = 'Auth providers';

describe('Access Control Auth providers', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_SCOPED_ACCESS_CONTROL_V2')) {
            this.skip();
        }
    });

    function visitAuthProviders() {
        cy.intercept('GET', authProvidersApi.list).as('GetAuthProviders');
        cy.visit(authProvidersUrl);
        cy.wait('@GetAuthProviders');
    }

    it('displays alert if no permission', () => {
        cy.intercept('GET', permissionsApi.mypermissions, {
            fixture: 'auth/mypermissionsMinimalAccess.json',
        }).as('GetMyPermissions');
        cy.visit(authProvidersUrl);
        cy.wait('@GetMyPermissions');

        cy.get(selectors.h1).should('have.text', h1);
        cy.get(selectors.navLink).should('not.exist');

        cy.get(selectors.h2).should('not.exist');

        cy.get(selectors.alertTitle).should(
            'contain', // instead of have.text because of "Info alert:" for screen reader
            'You do not have permission to view Access Control'
        );
    });

    it('list has breadcrumbs, headings, link, button, and table head cells', () => {
        visitAuthProviders();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h1}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("${h2}")`);

        cy.get(selectors.h1).should('have.text', h1);
        cy.get(selectors.navLinkCurrent).should('have.text', h2);

        cy.get(selectors.h2).should('have.text', h2);
        cy.get(selectors.list.addButton).should('have.text', 'Add auth provider');

        cy.get(`${selectors.list.th}:contains("Name")`);
        cy.get(`${selectors.list.th}:contains("Type")`);
        cy.get(`${selectors.list.th}:contains("Minimum access role")`);
        cy.get(`${selectors.list.th}:contains("Assigned rules")`);
    });

    it('add Auth0', () => {
        visitAuthProviders();

        const type = 'Auth0';

        cy.get(selectors.list.addButton).click();
        cy.get(`${selectors.list.authProviders.addDropdownItem}:contains("${type}")`).click();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h1}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(3):contains("Add auth provider")`);

        cy.get(selectors.h1).should('have.text', h1);
        cy.get(selectors.navLinkCurrent).should('have.text', h2);

        cy.get(selectors.h2).should('have.text', `Add new ${type} auth provider`);

        cy.get(selectors.form.inputName).should('be.enabled').should('have.attr', 'required');
        cy.get(selectors.form.authProvider.selectAuthProviderType)
            .should('be.disabled')
            .should('contain', type);

        cy.get(selectors.form.authProvider.auth0.inputAuth0Tenant)
            .should('be.enabled')
            .should('have.attr', 'required');
        cy.get(selectors.form.authProvider.auth0.inputClientID)
            .should('be.enabled')
            .should('have.attr', 'required');

        cy.get(selectors.form.saveButton).should('be.disabled');
        cy.get(selectors.form.cancelButton).should('be.enabled');
    });

    it('add OpenID Connect', () => {
        visitAuthProviders();

        const type = 'OpenID Connect';

        cy.get(selectors.list.addButton).click();
        cy.get(`${selectors.list.authProviders.addDropdownItem}:contains("${type}")`).click();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h1}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(3):contains("Add auth provider")`);

        cy.get(selectors.h1).should('have.text', h1);
        cy.get(selectors.navLinkCurrent).should('have.text', h2);

        cy.get(selectors.h2).should('have.text', `Add new ${type} auth provider`);

        cy.get(selectors.form.inputName).should('be.enabled').should('have.attr', 'required');
        cy.get(selectors.form.authProvider.selectAuthProviderType)
            .should('be.disabled')
            .should('contain', type);

        const {
            selectCallbackMode,
            selectCallbackModeItem,
            inputIssuer,
            inputClientID,
            inputClientSecret,
            checkboxDoNotUseClientSecret,
        } = selectors.form.authProvider.oidc;

        cy.get(selectCallbackMode)
            .should('be.enabled')
            .should('contain', 'Auto-select (recommended)');
        cy.get(inputIssuer).should('be.enabled').should('have.attr', 'required');
        cy.get(inputClientID).should('be.enabled').should('have.attr', 'required');
        cy.get(inputClientSecret).should('be.enabled').should('have.attr', 'required');
        cy.get(checkboxDoNotUseClientSecret).should('be.enabled').should('not.be.checked');

        cy.get(selectors.form.saveButton).should('be.disabled');
        cy.get(selectors.form.cancelButton).should('be.enabled');

        cy.get(selectCallbackMode).click();
        cy.get(`${selectCallbackModeItem}:contains("Fragment")`).click();
        cy.get(inputClientSecret).should('be.disabled');
        // cy.get(checkboxDoNotUseClientSecret).should('be.disabled'); // TODO classic test asserts is should be disabled

        cy.get(selectCallbackMode).click();
        cy.get(`${selectCallbackModeItem}:contains("HTTP POST")`).click();
        cy.get(checkboxDoNotUseClientSecret).check();
        cy.get(inputClientSecret).should('be.disabled').should('not.have.attr', 'required');
    });

    it('edits OpenID Connect with a client secret without losing the value', () => {
        cy.intercept('GET', authProvidersApi.list, {
            fixture: 'auth/authProvidersWithClientSecret.json',
        }).as('GetAuthProviders');
        cy.intercept('GET', groupsApi.list, {
            fixture: 'auth/groupsWithClientSecret.json',
        }).as('GetGroups'); // to compute default access role
        cy.intercept('PUT', '/v1/authProviders/auth-provider-1', {
            body: {},
        }).as('PutAuthProvider');

        const id = 'auth-provider-1';
        cy.visit(`${authProvidersUrl}/${id}`);
        cy.wait(['@GetAuthProviders', '@GetGroups']);

        const {
            inputIssuer,
            inputClientSecret,
            checkboxDoNotUseClientSecret,
        } = selectors.form.authProvider.oidc;

        cy.get(inputClientSecret).should('be.disabled').should('have.value', ''); // TODO was ****** in classic test
        // cy.get(checkboxDoNotUseClientSecret).should('be.disabled'); // TODO see above and should it be checked or not?

        cy.get(selectors.form.editButton).click();

        // cy.get(inputClientSecret).should('be.enabled');
        cy.get(checkboxDoNotUseClientSecret).should('be.enabled');

        cy.get(inputIssuer).clear().type('irrelevant-updated');
        /*
        cy.get(selectors.form.saveButton).click(); // TODO disabled, because not valid?
        cy.wait('@PutAuthProvider');

        cy.get(inputClientSecret).should('be.disabled').should('have.value', ''); // TODO was ****** in classic test
        // cy.get(checkboxDoNotUseClientSecret).should('be.disabled'); // TODO see above and should it be checked or not?
        */
    });

    it('add SAML 2.0', () => {
        visitAuthProviders();

        const type = 'SAML 2.0';

        cy.get(selectors.list.addButton).click();
        cy.get(`${selectors.list.authProviders.addDropdownItem}:contains("${type}")`).click();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h1}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(3):contains("Add auth provider")`);

        cy.get(selectors.h1).should('have.text', h1);
        cy.get(selectors.navLinkCurrent).should('have.text', h2);

        cy.get(selectors.h2).should('have.text', `Add new ${type} auth provider`);

        cy.get(selectors.form.inputName).should('be.enabled').should('have.attr', 'required');
        cy.get(selectors.form.authProvider.selectAuthProviderType)
            .should('be.disabled')
            .should('contain', type);

        cy.get(selectors.form.authProvider.saml.inputServiceProviderIssuer)
            .should('be.enabled')
            .should('have.attr', 'required');
        cy.get(selectors.form.authProvider.saml.selectConfiguration)
            .should('be.enabled')
            .should('contain', 'Option 1: Dynamic configuration');
        cy.get(selectors.form.authProvider.saml.inputMetadataURL)
            .should('be.enabled')
            .should('have.attr', 'required');

        cy.get(selectors.form.saveButton).should('be.disabled');
        cy.get(selectors.form.cancelButton).should('be.enabled');
    });

    it('add User Certificates', () => {
        visitAuthProviders();

        const type = 'User Certificates';

        cy.get(selectors.list.addButton).click();
        cy.get(`${selectors.list.authProviders.addDropdownItem}:contains("${type}")`).click();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h1}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(3):contains("Add auth provider")`);

        cy.get(selectors.h1).should('have.text', h1);
        cy.get(selectors.navLinkCurrent).should('have.text', h2);

        cy.get(selectors.h2).should('have.text', `Add new ${type} auth provider`);

        cy.get(selectors.form.inputName).should('be.enabled').should('have.attr', 'required');
        cy.get(selectors.form.authProvider.selectAuthProviderType)
            .should('be.disabled')
            .should('contain', type);

        cy.get(selectors.form.authProvider.userpki.textareaCertificates)
            .should('be.enabled')
            .should('have.attr', 'required');

        cy.get(selectors.form.saveButton).should('be.disabled');
        cy.get(selectors.form.cancelButton).should('be.enabled');
    });

    it('add Google IAP', () => {
        visitAuthProviders();

        const type = 'Google IAP';

        cy.get(selectors.list.addButton).click();
        cy.get(`${selectors.list.authProviders.addDropdownItem}:contains("${type}")`).click();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h1}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(3):contains("Add auth provider")`);

        cy.get(selectors.h1).should('have.text', h1);
        cy.get(selectors.navLinkCurrent).should('have.text', h2);

        cy.get(selectors.h2).should('have.text', `Add new ${type} auth provider`);

        cy.get(selectors.form.inputName).should('be.enabled').should('have.attr', 'required');
        cy.get(selectors.form.authProvider.selectAuthProviderType)
            .should('be.disabled')
            .should('contain', type);

        cy.get(selectors.form.authProvider.iap.inputAudience)
            .should('be.enabled')
            .should('have.attr', 'required');

        cy.get(selectors.form.saveButton).should('be.disabled');
        cy.get(selectors.form.cancelButton).should('be.enabled');
    });
});
