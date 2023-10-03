import sampleCert from '../../helpers/sampleCert';
import { generateNameWithDate, getInputByLabel } from '../../helpers/formHelpers';
import updateMinimumAccessRoleRequest from '../../fixtures/auth/updateMinimumAccessRole.json';

import withAuth from '../../helpers/basicAuth';
import { assertCannotFindThePage } from '../../helpers/visit';
import { checkInviteUsersModal } from '../../helpers/inviteUsers';

import {
    assertAccessControlEntitiesPage,
    assertAccessControlEntityDoesNotExist,
    assertAccessControlEntityPage,
    authProvidersAlias,
    authProvidersAliasForDELETE,
    authProvidersAliasForPOST,
    authProvidersAliasForPUT,
    authProvidersKey as entitiesKey,
    clickConfirmationToDeleteAuthProvider,
    clickRowActionMenuItemInTable,
    groupsAlias,
    groupsBatchAliasForPOST,
    rolesAlias,
    saveCreatedAuthProvider,
    saveUpdatedAuthProvider,
    visitAccessControlEntities,
    visitAccessControlEntitiesWithStaticResponseForPermissions,
    visitAccessControlEntity,
} from './accessControl.helpers';
import { selectors, accessModalSelectors } from './accessControl.selectors';

describe('Access Control Auth providers', () => {
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

    it('list has table head cells', () => {
        const staticResponseMap = {
            [authProvidersAlias]: {
                fixture: 'auth/authProviders-id1-id2-id3.json',
            },
        };
        visitAccessControlEntities(entitiesKey, staticResponseMap);

        cy.get('th:contains("Name")');
        cy.get('th:contains("Origin")');
        cy.get('th:contains("Type")');
        cy.get('th:contains("Minimum access role")');
        cy.get('th:contains("Assigned rules")');
        cy.get('th[aria-label="Row actions"]');
    });

    it('add Auth0', () => {
        visitAccessControlEntities(entitiesKey);

        const type = 'Auth0';

        cy.get('button:contains("Create auth provider")').click();
        cy.get(`${selectors.list.authProviders.createDropdownItem}:contains("${type}")`).click();

        assertAccessControlEntityPage(entitiesKey);

        cy.get('h2').should('have.text', `Create ${type} provider`);
        cy.get(`li.pf-c-breadcrumb__item:nth-child(2):contains("Create ${type} provider")`);

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
        cy.get(selectors.form.cancelButton).click();

        assertAccessControlEntitiesPage(entitiesKey);
    });

    it('add OpenID Connect', () => {
        visitAccessControlEntities(entitiesKey);

        const type = 'OpenID Connect';

        cy.get('button:contains("Create auth provider")').click();
        cy.get(`${selectors.list.authProviders.createDropdownItem}:contains("${type}")`).click();

        assertAccessControlEntityPage(entitiesKey);

        cy.get('h2').should('have.text', `Create ${type} provider`);
        cy.get(`li.pf-c-breadcrumb__item:nth-child(2):contains("Create ${type} provider")`);

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

        cy.get(selectors.form.saveButton).should('be.disabled');
        cy.get(selectors.form.cancelButton).click();

        assertAccessControlEntitiesPage(entitiesKey);
    });

    it('edits OpenID Connect with a client secret without losing the value', () => {
        const entityId = 'auth-provider-1'; // corresponds to fixtures
        const staticResponseMapForAuthProvider = {
            [authProvidersAlias]: {
                fixture: 'auth/authProvidersWithClientSecret.json',
            },
            [groupsAlias]: {
                fixture: 'auth/groupsWithClientSecret.json', // to compute default access role
            },
        };
        visitAccessControlEntity(entitiesKey, entityId, staticResponseMapForAuthProvider);

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

        // Mock responses to save updated auth provider.
        const staticResponseMapForUpdatedAuthProvider = {
            [authProvidersAliasForPUT]: {
                body: {},
            },
            [groupsBatchAliasForPOST]: {
                body: {},
            },
        };
        saveUpdatedAuthProvider(entityId, staticResponseMapForUpdatedAuthProvider);

        cy.get(inputClientSecret)
            .should('be.disabled')
            .should('have.value', '')
            .should('have.attr', 'placeholder', '*****');
        cy.get(checkboxDoNotUseClientSecret).should('be.disabled').should('not.be.checked');
    });

    it('edit OpenID Connect minimum access role', () => {
        const entityId = 'auth-provider-1'; // corresponds to fixtures
        const staticResponseMapForAuthProvider = {
            [authProvidersAlias]: {
                fixture: 'auth/authProvidersWithClientSecret.json',
            },
            [groupsAlias]: {
                fixture: 'auth/groupsWithClientSecret.json', // to compute default access role
            },
        };
        visitAccessControlEntity(entitiesKey, entityId, staticResponseMapForAuthProvider);

        const { selectMinimumAccessRole, selectMinimumAccessRoleItem } =
            selectors.form.minimumAccessRole;

        cy.get(selectors.form.editButton).click();

        cy.get(selectMinimumAccessRole).should('be.enabled').should('contain', 'Admin');
        cy.get(selectMinimumAccessRole).click();
        cy.get(`${selectMinimumAccessRoleItem}:contains("Analyst")`).click();

        // Mock responses to save updated auth provider.
        const staticResponseMapForUpdatedAuthProvider = {
            [authProvidersAliasForPUT]: {
                body: {},
            },
            [groupsBatchAliasForPOST]: {
                body: {},
            },
        };
        saveUpdatedAuthProvider(entityId, staticResponseMapForUpdatedAuthProvider).then(
            ([, { request }]) => {
                expect(request.body).to.deep.equal(
                    updateMinimumAccessRoleRequest,
                    `request: ${JSON.stringify(request.body)} expected: ${JSON.stringify(
                        updateMinimumAccessRoleRequest
                    )}`
                );
            }
        );

        cy.get(selectMinimumAccessRole).should('contain', 'Analyst');
    });

    it('add SAML 2.0', () => {
        visitAccessControlEntities(entitiesKey);

        const type = 'SAML 2.0';

        cy.get('button:contains("Create auth provider")').click();
        cy.get(`${selectors.list.authProviders.createDropdownItem}:contains("${type}")`).click();

        assertAccessControlEntityPage(entitiesKey);

        cy.get('h2').should('have.text', `Create ${type} provider`);
        cy.get(`li.pf-c-breadcrumb__item:nth-child(2):contains("Create ${type} provider")`);

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
        cy.get(selectors.form.cancelButton).click();

        assertAccessControlEntitiesPage(entitiesKey);
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

        visitAccessControlEntities(entitiesKey);

        const type = 'User Certificates';

        cy.get('button:contains("Create auth provider")').click();
        cy.get(`${selectors.list.authProviders.createDropdownItem}:contains("${type}")`).click();

        assertAccessControlEntityPage(entitiesKey);

        cy.get('h2').should('have.text', `Create ${type} provider`);
        cy.get(`li.pf-c-breadcrumb__item:nth-child(2):contains("Create ${type} provider")`);

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

        const staticResponseMapForCreatedAuthProvider = {
            [authProvidersAliasForPOST]: {
                body: mockUserCertResponse,
            },
            [groupsBatchAliasForPOST]: {
                body: {},
            },
        };
        saveCreatedAuthProvider(staticResponseMapForCreatedAuthProvider);

        assertAccessControlEntitiesPage(entitiesKey);
        cy.location('search').should('eq', '');
    });

    it('add Google IAP', () => {
        visitAccessControlEntities(entitiesKey);

        const type = 'Google IAP';

        cy.get('button:contains("Create auth provider")').click();
        cy.get(`${selectors.list.authProviders.createDropdownItem}:contains("${type}")`).click();

        assertAccessControlEntityPage(entitiesKey);

        cy.get('h2').should('have.text', `Create ${type} provider`);
        cy.get(`li.pf-c-breadcrumb__item:nth-child(2):contains("Create ${type} provider")`);

        cy.get(selectors.form.inputName).should('be.enabled').should('have.attr', 'required');
        cy.get(selectors.form.authProvider.selectAuthProviderType)
            .should('be.disabled')
            .should('contain', type);

        cy.get(selectors.form.authProvider.iap.inputAudience)
            .should('be.enabled')
            .should('have.attr', 'required');

        cy.get(selectors.form.saveButton).should('be.disabled');
        cy.get(selectors.form.cancelButton).click();

        assertAccessControlEntitiesPage(entitiesKey);
    });

    describe('empty state', () => {
        it('should show a confirmation before deleting a provider', () => {
            const entityName = 'auth-provider-1'; // corresponds to fixture
            const staticResponseMap = {
                [authProvidersAlias]: {
                    fixture: 'auth/authProviders-id1.json',
                },
            };
            visitAccessControlEntities(entitiesKey, staticResponseMap);

            clickRowActionMenuItemInTable(entityName, 'Delete auth provider');

            cy.get(accessModalSelectors.title);
            cy.get(accessModalSelectors.body);
            cy.get(accessModalSelectors.delete);
            cy.get(accessModalSelectors.cancel).click();

            assertAccessControlEntitiesPage(entitiesKey);
        });

        it('should show empty state after deleting the last provider', () => {
            const entityId = 'authProvider-id1'; // corresponds to fixture
            const entityName = 'auth-provider-1'; // corresponds to fixture
            const staticResponseMap = {
                [authProvidersAlias]: {
                    fixture: 'auth/authProviders-id1.json',
                },
            };
            visitAccessControlEntities(entitiesKey, staticResponseMap);

            clickRowActionMenuItemInTable(entityName, 'Delete auth provider');

            const staticResponseMapToDeleteAuthProvider = {
                [authProvidersAliasForDELETE]: {
                    body: {},
                },
                [authProvidersAlias]: {
                    body: { authProviders: [] }, // empty array like nothing is left
                },
            };
            clickConfirmationToDeleteAuthProvider(entityId, staticResponseMapToDeleteAuthProvider);

            cy.get('.pf-c-empty-state__content:contains("No auth providers")');
        });
    });

    it('displays message instead of form if entity id does not exist', () => {
        const entityId = 'bogus';
        visitAccessControlEntity(entitiesKey, entityId);

        assertAccessControlEntityDoesNotExist(entitiesKey);
    });
});

describe('Invite users', () => {
    withAuth();

    it('should have a trigger for opening the Invite users modal in the Auth Providers table header', () => {
        const staticResponseMap = {
            [authProvidersAlias]: {
                fixture: 'auth/authProviders-id1-id3.json',
            },
            [rolesAlias]: {
                fixture: 'auth/roles.json',
            },
        };
        visitAccessControlEntities(entitiesKey, staticResponseMap);

        cy.get('button:contains("Invite users")').click();

        checkInviteUsersModal();
    });

    it('should warn if there are no auth providers available', () => {
        const staticResponseMap = {
            [authProvidersAlias]: {
                fixture: 'auth/authProviders-empty.json',
            },
            [rolesAlias]: {
                fixture: 'auth/roles.json',
            },
        };
        visitAccessControlEntities(entitiesKey, staticResponseMap);

        cy.get('button:contains("Invite users")').click();

        cy.get('.pf-c-alert__title:contains("No auth providers are available")');
        cy.get('.pf-c-modal-box__body .pf-m-warning a:contains("Access Control")').click();

        assertAccessControlEntitiesPage(entitiesKey);
    });
});
