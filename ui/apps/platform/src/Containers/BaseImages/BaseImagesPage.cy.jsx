import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import { PublicConfigContext } from 'hooks/usePublicConfig';
import { UserPermissionContext } from 'hooks/usePermissions';

import BaseImagesPage from './BaseImagesPage';

const mockPublicConfigValue = {
    publicConfig: { telemetry: { enabled: false } },
    isLoadingPublicConfig: false,
    error: undefined,
    refetchPublicConfig: () => {},
};

const mockPermissions = {
    userRolePermissions: { resourceToAccess: { ImageAdministration: 'READ_WRITE_ACCESS' } },
    isLoadingPermissions: false,
};

const baseImage = {
    id: 'base-image-id',
    baseImageRepoPath: 'docker.io/library/ubuntu',
    baseImageTagPattern: '22.04',
    user: { id: 'user-id', name: 'admin' },
};

function mountPage() {
    cy.mount(
        <ComponentTestProvider>
            <PublicConfigContext.Provider value={mockPublicConfigValue}>
                <UserPermissionContext.Provider value={mockPermissions}>
                    <BaseImagesPage />
                </UserPermissionContext.Provider>
            </PublicConfigContext.Provider>
        </ComponentTestProvider>
    );
}

function getToastTitle(title) {
    return cy.get('ul.pf-v6-c-alert-group.pf-m-toast .pf-v6-c-alert__title').contains(title);
}

function openRowAction(actionName) {
    cy.findByRole('button', { name: 'Kebab toggle' }).click();
    cy.findByRole('menuitem', { name: actionName }).click();
}

describe(Cypress.spec.relative, () => {
    beforeEach(() => {
        cy.intercept('GET', '/v2/baseimages', {
            statusCode: 200,
            body: { baseImageReferences: [baseImage] },
        }).as('getBaseImages');
    });

    it('should show a toast on the page after adding a base image', () => {
        cy.intercept('POST', '/v2/baseimages', {
            statusCode: 200,
            body: { baseImageReference: { ...baseImage, id: 'new-base-image-id' } },
        }).as('addBaseImage');

        mountPage();
        cy.wait('@getBaseImages');

        cy.findByRole('button', { name: 'Add base image' }).click();
        cy.get('input#baseImagePath').type('docker.io/library/alpine:3.18');
        cy.findByRole('button', { name: 'Save' }).click();

        cy.wait('@addBaseImage');
        cy.findByRole('dialog').should('not.exist');
        getToastTitle('Base image successfully added').should('be.visible');
    });

    it('should show a toast on the page after updating a tag pattern', () => {
        cy.intercept('PUT', '/v2/baseimages/base-image-id', { statusCode: 200, body: {} }).as(
            'updateBaseImage'
        );

        mountPage();
        cy.wait('@getBaseImages');

        openRowAction('Edit tag pattern');
        cy.get('input#baseImageTagPattern').clear();
        cy.get('input#baseImageTagPattern').type('24.*');
        cy.findByRole('button', { name: 'Save' }).click();

        cy.wait('@updateBaseImage');
        cy.findByRole('dialog').should('not.exist');
        getToastTitle('Base image tag pattern successfully updated').should('be.visible');
    });

    it('should show a toast on the page after deleting a base image', () => {
        cy.intercept('DELETE', '/v2/baseimages/base-image-id', { statusCode: 200, body: {} }).as(
            'deleteBaseImage'
        );

        mountPage();
        cy.wait('@getBaseImages');

        openRowAction('Delete base image');
        cy.findByRole('dialog').within(() => {
            cy.findByRole('button', { name: 'Delete' }).click();
        });

        cy.wait('@deleteBaseImage');
        cy.findByRole('dialog').should('not.exist');
        getToastTitle('Base image successfully deleted').should('be.visible');
    });
});
