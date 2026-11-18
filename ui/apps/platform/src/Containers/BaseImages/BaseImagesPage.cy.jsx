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

    it('should reopen the add modal clean after adding a base image', () => {
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
        cy.wait('@getBaseImages'); // the table refetches after the save
        cy.findByRole('dialog').should('not.exist');
        cy.findByText('Base image successfully added').should('not.exist');

        cy.findByRole('button', { name: 'Add base image' }).click();
        cy.findByRole('dialog').should('be.visible');
        cy.get('input#baseImagePath').should('have.value', '');
        cy.findByText('Base image successfully added').should('not.exist');
    });

    it('should reopen the edit modal clean after updating a tag pattern', () => {
        cy.intercept('PUT', '/v2/baseimages/base-image-id', { statusCode: 200, body: {} }).as(
            'updateBaseImage'
        );

        mountPage();
        cy.wait('@getBaseImages');

        openRowAction('Edit tag pattern');
        cy.get('input#baseImageTagPattern').clear();
        cy.get('input#baseImageTagPattern').type('24.*');
        cy.intercept('GET', '/v2/baseimages', {
            statusCode: 200,
            body: { baseImageReferences: [{ ...baseImage, baseImageTagPattern: '24.*' }] },
        }).as('getUpdatedBaseImages');
        cy.findByRole('button', { name: 'Save' }).click();

        cy.wait('@updateBaseImage');
        cy.wait('@getUpdatedBaseImages');
        cy.findByRole('dialog').should('not.exist');
        cy.findByText('Base image tag pattern successfully updated').should('not.exist');

        openRowAction('Edit tag pattern');
        cy.findByRole('dialog').should('be.visible');
        cy.get('input#baseImageTagPattern').should('have.value', '24.*');
        cy.findByText('Base image tag pattern successfully updated').should('not.exist');
    });

    it('should reopen the add modal without the error from a failed save', () => {
        cy.intercept('POST', '/v2/baseimages', {
            statusCode: 500,
            body: { message: 'Internal server error' },
        }).as('addBaseImageError');

        mountPage();
        cy.wait('@getBaseImages');

        cy.findByRole('button', { name: 'Add base image' }).click();
        cy.get('input#baseImagePath').type('docker.io/library/alpine:3.18');
        cy.findByRole('button', { name: 'Save' }).click();

        cy.wait('@addBaseImageError');
        cy.findByText('Error adding base image').should('be.visible');
        cy.findByRole('button', { name: 'Cancel' }).click();
        cy.findByRole('dialog').should('not.exist');

        cy.findByRole('button', { name: 'Add base image' }).click();
        cy.findByRole('dialog').should('be.visible');
        cy.get('input#baseImagePath').should('have.value', '');
        cy.findByText('Error adding base image').should('not.exist');
    });
});
