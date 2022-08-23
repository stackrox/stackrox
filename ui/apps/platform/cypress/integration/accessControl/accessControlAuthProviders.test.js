import {
    authProvidersUrl,
    selectors,
    accessModalSelectors,
} from '../../constants/AccessControlPage';
import * as api from '../../constants/apiEndpoints';
import sampleCert from '../../helpers/sampleCert';
import { generateNameWithDate, getInputByLabel } from '../../helpers/formHelpers';
import { visit, visitWithPermissions } from '../../helpers/visit';
import updateMinimumAccessRoleRequest from '../../fixtures/auth/updateMinimumAccessRole.json';

import withAuth from '../../helpers/basicAuth';

const h1 = 'Access Control';
const h2 = 'Auth providers';

const routeMatcherMap = {
    authProviders: {
        method: 'GET',
        url: api.auth.authProviders,
    },
    roles: {
        method: 'GET',
        url: api.roles.list,
    },
    groups: {
        method: 'GET',
        url: api.groups.list,
    },
};

function visitAuthProviders(staticResponseMap) {
    visit(authProvidersUrl, { routeMatcherMap }, staticResponseMap);

    cy.get(selectors.breadcrumbNav).should('not.exist');
    cy.get(`${selectors.h1}:contains("${h1}")`);
    cy.get(`${selectors.navLinkCurrent}:contains("${h2}")`);
    cy.get(selectors.list.createButton).should('have.text', 'Create auth provider');
    cy.contains(selectors.h2, /^\d+ results? found$/).should('exist');
}

function visitAuthProviderById(id, staticResponseMap) {
    visit(`${authProvidersUrl}/${id}`, { routeMatcherMap }, staticResponseMap);
}

describe('Access Control Auth providers', () => {
    withAuth();

    it('displays alert if no permission', () => {
        const permissionsStaticResponse = {
            fixture: 'auth/mypermissionsMinimalAccess.json',
        };
        visitWithPermissions(authProvidersUrl, permissionsStaticResponse);

        cy.get(`${selectors.h1}:contains("${h1}")`);
        cy.get(selectors.navLink).should('not.exist');
        cy.get(selectors.h2).should('not.exist');

        cy.get(selectors.alertTitle).should(
            'contain', // instead of have.text because of "Info alert:" for screen reader
            'You do not have permission to view Access Control'
        );
    });

    it('list has table head cells', () => {
        const staticResponseMap = {
            authProviders: {
                fixture: 'auth/authProviders-id1-id2-id3.json',
            },
        };
        visitAuthProviders(staticResponseMap);

        cy.get(`${selectors.list.th}:contains("Name")`);
        cy.get(`${selectors.list.th}:contains("Type")`);
        cy.get(`${selectors.list.th}:contains("Minimum access role")`);
        cy.get(`${selectors.list.th}:contains("Assigned rules")`);
    });

    it('add Auth0', () => {
        visitAuthProviders();

        const type = 'Auth0';

        cy.get(selectors.list.createButton).click();
        cy.get(`${selectors.list.authProviders.createDropdownItem}:contains("${type}")`).click();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("Create ${type} provider")`);

        cy.get(selectors.h1).should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');

        cy.get(selectors.h2).should('have.text', `Create ${type} provider`);

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

        cy.get(selectors.list.createButton).click();
        cy.get(`${selectors.list.authProviders.createDropdownItem}:contains("${type}")`).click();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("Create ${type} provider")`);

        cy.get(selectors.h1).should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');

        cy.get(selectors.h2).should('have.text', `Create ${type} provider`);

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
        const staticResponseMap = {
            authProviders: {
                fixture: 'auth/authProvidersWithClientSecret.json',
            },
            groups: {
                fixture: 'auth/groupsWithClientSecret.json', // to compute default access role
            },
        };

        cy.intercept('PUT', '/v1/authProviders/auth-provider-1', {
            body: {},
        }).as('PutAuthProvider');
        cy.intercept('POST', api.groups.batch, {
            body: {},
        }).as('PostGroupsBatch');

        const id = 'auth-provider-1';
        visitAuthProviderById(id, staticResponseMap);

        const { inputIssuer, inputClientSecret, checkboxDoNotUseClientSecret } =
            selectors.form.authProvider.oidc;

        cy.get(inputClientSecret)
            .should('be.disabled')
            .should('have.value', '')
            .should('have.attr', 'placeholder', '*****');
        cy.get(checkboxDoNotUseClientSecret).should('be.disabled').should('not.be.checked');

        cy.get(selectors.form.editButton).click();

        cy.get(inputClientSecret)
            .should('be.enabled')
            .should('have.value', '')
            .should('not.have.attr', 'placeholder', '*****');
        cy.get(checkboxDoNotUseClientSecret).should('be.enabled').should('not.be.checked');

        cy.get(inputIssuer).clear().type('irrelevant-updated');

        cy.get(selectors.form.saveButton).click();
        cy.wait(['@PutAuthProvider', '@PostGroupsBatch']);

        cy.get(inputClientSecret)
            .should('be.disabled')
            .should('have.value', '')
            .should('have.attr', 'placeholder', '*****');
        cy.get(checkboxDoNotUseClientSecret).should('be.disabled').should('not.be.checked');
    });

    it('edit OpenID Connect minimum access role', () => {
        const staticResponseMap = {
            authProviders: {
                fixture: 'auth/authProvidersWithClientSecret.json',
            },
            groups: {
                fixture: 'auth/groupsWithClientSecret.json', // to compute default access role
            },
        };

        cy.intercept('PUT', '/v1/authProviders/auth-provider-1', {
            body: {},
        }).as('PutAuthProvider');
        cy.intercept('POST', api.groups.batch, (req) => {
            expect(req.body).to.deep.equal(
                updateMinimumAccessRoleRequest,
                `request: ${JSON.stringify(req.body)} expected: ${JSON.stringify(
                    updateMinimumAccessRoleRequest
                )}`
            );
            req.body = {};
        }).as('PostGroupsBatch');

        const id = 'auth-provider-1';
        visitAuthProviderById(id, staticResponseMap);

        const { selectMinimumAccessRole, selectMinimumAccessRoleItem } =
            selectors.form.minimumAccessRole;

        cy.get(selectors.form.editButton).click();

        cy.get(selectMinimumAccessRole).should('be.enabled').should('contain', 'Admin');
        cy.get(selectMinimumAccessRole).click();
        cy.get(`${selectMinimumAccessRoleItem}:contains("Analyst")`).click();

        cy.get(selectors.form.saveButton).click();
        cy.wait(['@PutAuthProvider', '@PostGroupsBatch']);

        cy.get(selectMinimumAccessRole).should('contain', 'Analyst');
    });

    it('add SAML 2.0', () => {
        visitAuthProviders();

        const type = 'SAML 2.0';

        cy.get(selectors.list.createButton).click();
        cy.get(`${selectors.list.authProviders.createDropdownItem}:contains("${type}")`).click();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("Create ${type} provider")`);

        cy.get(selectors.h1).should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');

        cy.get(selectors.h2).should('have.text', `Create ${type} provider`);

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
        const newProviderName = generateNameWithDate('User Cert Test Provide');

        const mockUserCertResponse = {
            id: '21b1003e-8f24-447f-a92e-0f8ff4a1274e',
            name: newProviderName,
            type: 'userpki',
            uiEndpoint: 'localhost:3000',
            enabled: true,
            config: {
                keys: '-----BEGIN CERTIFICATE-----\nMIICEjCCAXsCAg36MA0GCSqGSIb3DQEBBQUAMIGbMQswCQYDVQQGEwJKUDEOMAwG\nA1UECBMFVG9reW8xEDAOBgNVBAcTB0NodW8ta3UxETAPBgNVBAoTCEZyYW5rNERE\nMRgwFgYDVQQLEw9XZWJDZXJ0IFN1cHBvcnQxGDAWBgNVBAMTD0ZyYW5rNEREIFdl\nYiBDQTEjMCEGCSqGSIb3DQEJARYUc3VwcG9ydEBmcmFuazRkZC5jb20wHhcNMTIw\nODIyMDUyNjU0WhcNMTcwODIxMDUyNjU0WjBKMQswCQYDVQQGEwJKUDEOMAwGA1UE\nCAwFVG9reW8xETAPBgNVBAoMCEZyYW5rNEREMRgwFgYDVQQDDA93d3cuZXhhbXBs\nZS5jb20wXDANBgkqhkiG9w0BAQEFAANLADBIAkEAm/xmkHmEQrurE/0re/jeFRLl\n8ZPjBop7uLHhnia7lQG/5zDtZIUC3RVpqDSwBuw/NTweGyuP+o8AG98HxqxTBwID\nAQABMA0GCSqGSIb3DQEBBQUAA4GBABS2TLuBeTPmcaTaUW/LCB2NYOy8GMdzR1mx\n8iBIu2H6/E2tiY3RIevV2OW61qY2/XRQg7YPxx3ffeUugX9F4J/iPnnu1zAxxyBy\n2VguKv4SWjRFoRkIfIlHX0qVviMhSlNy2ioFLy7JcPZb+v3ftDGywUqcBiVDoea0\nHn+GmxZA\n-----END CERTIFICATE-----',
            },
            loginUrl: '/sso/login/21b1003e-8f24-447f-a92e-0f8ff4a1274e',
            validated: false,
            extraUiEndpoints: [],
            active: false,
        };

        visitAuthProviders();

        const type = 'User Certificates';

        cy.get(selectors.list.createButton).click();
        cy.get(`${selectors.list.authProviders.createDropdownItem}:contains("${type}")`).click();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("Create ${type} provider")`);

        cy.get(selectors.h1).should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');

        cy.get(selectors.h2).should('have.text', `Create ${type} provider`);

        getInputByLabel('Name').should('be.enabled').should('have.attr', 'required');
        cy.get(selectors.form.authProvider.selectAuthProviderType)
            .should('be.disabled')
            .should('contain', type);

        getInputByLabel('CA certificate(s) (PEM)')
            .should('be.enabled')
            .should('have.attr', 'required');

        cy.get(selectors.form.saveButton).should('be.disabled');
        cy.get(selectors.form.cancelButton).should('be.enabled');

        getInputByLabel('Name').type(newProviderName, { delay: 1 });
        getInputByLabel('CA certificate(s) (PEM)').type(sampleCert, {
            delay: 1,
        });

        cy.intercept('POST', api.auth.authProviders, { body: mockUserCertResponse }).as(
            'CreateAuthProvider'
        ); // mock POST means that the list is empty after save

        cy.get(selectors.form.saveButton).should('be.enabled').click();

        cy.wait('@CreateAuthProvider'); // wait for POST to finish
        cy.wait('@authProviders'); // wait for GET to finish, which means redirect back to list page
        cy.location('pathname').should('eq', '/main/access-control/auth-providers');
        cy.location('search').should('eq', '');
    });

    it('add Google IAP', () => {
        visitAuthProviders();

        const type = 'Google IAP';

        cy.get(selectors.list.createButton).click();
        cy.get(`${selectors.list.authProviders.createDropdownItem}:contains("${type}")`).click();

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2):contains("Create ${type} provider")`);

        cy.get(selectors.h1).should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');

        cy.get(selectors.h2).should('have.text', `Create ${type} provider`);

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

    describe('empty state', () => {
        it('should show a confirmation before deleting a provider', () => {
            const staticResponseMap = {
                authProviders: {
                    fixture: 'auth/authProviders-id1.json',
                },
            };
            visitAuthProviders(staticResponseMap);

            cy.get(selectors.list.authProviders.tdActions).click();

            cy.get(selectors.list.authProviders.deleteActionItem).click();

            cy.get(accessModalSelectors.title);
            cy.get(accessModalSelectors.body);
            cy.get(accessModalSelectors.delete);
            cy.get(accessModalSelectors.cancel).click();

            cy.get(selectors.list.authProviders.dataRows);
        });

        it('should show empty state after deleting the last provider', () => {
            const staticResponseMap = {
                authProviders: {
                    fixture: 'auth/authProviders-id1.json',
                },
            };
            visitAuthProviders(staticResponseMap);

            const id = 'authProvider-id1';
            cy.intercept('DELETE', `${api.auth.authProviders}/${id}`, {}).as('DeleteAuthProvider');
            cy.log(selectors.list);
            cy.get(selectors.list.authProviders.tdActions).click();

            cy.get(selectors.list.authProviders.deleteActionItem).click();

            // Mock now with empty list of providers like nothing is left.
            // Same alias as in staticResponseMap above.
            cy.intercept('GET', api.auth.authProviders, { authProviders: [] }).as('authProviders');
            cy.get(accessModalSelectors.delete).click();
            cy.wait(['@DeleteAuthProvider', '@authProviders']);

            cy.get(selectors.list.authProviders.emptyState);
        });
    });

    it('displays message instead of form if entity id does not exist', () => {
        const id = 'bogus';
        visitAuthProviderById(id);

        cy.get(`${selectors.breadcrumbItem}:nth-child(1):contains("${h2}")`);
        cy.get(`${selectors.breadcrumbItem}:nth-child(2)`).should('not.exist');

        cy.get(selectors.h1).should('not.exist');
        cy.get(selectors.navLinkCurrent).should('not.exist');
        cy.get(selectors.h2).should('not.exist');

        cy.get(selectors.notFound.title).should('have.text', 'Auth provider does not exist');
        cy.get(selectors.notFound.a)
            .should('have.text', h2)
            .should('have.attr', 'href', authProvidersUrl);
    });
});
