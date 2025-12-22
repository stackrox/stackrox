import withAuth from '../../helpers/basicAuth';
import { interceptAndOverrideFeatureFlags } from '../../helpers/request';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    addBaseImage,
    baseImagesAlias,
    basePath,
    deleteBaseImage,
    openAddModal,
    routeMatcherMapForGET,
    visitBaseImages,
    visitBaseImagesFromLeftNav,
} from './baseImages.helpers';
import { selectors } from './baseImages.selectors';

describe('Base Images', () => {
    withAuth();

    beforeEach(() => {
        interceptAndOverrideFeatureFlags({ ROX_BASE_IMAGE_DETECTION: true });
    });

    describe('Page access and navigation', () => {
        it('should navigate to Base Images page from left nav', () => {
            visitBaseImagesFromLeftNav();

            cy.get(selectors.pageTitle).should('be.visible');
            cy.get(selectors.pageDescription).should('be.visible');
            cy.get(selectors.table).should('exist');
        });

        it('should have correct page title', () => {
            visitBaseImages();

            cy.title().should('match', getRegExpForTitleWithBranding('Base Images'));
        });

        it('should display table with correct headers', () => {
            visitBaseImages();

            cy.get(selectors.tableHeader.baseImagePath).should('be.visible');
            cy.get(selectors.tableHeader.addedBy).should('be.visible');
        });
    });

    describe('Add base image', () => {
        it('should open add modal when clicking Add button', () => {
            visitBaseImages();

            openAddModal();

            cy.get(selectors.addModal.input).should('be.visible');
            cy.get(selectors.addModal.saveButton).should('be.disabled');
        });

        it('should show validation error for empty input', () => {
            visitBaseImages();

            openAddModal();
            cy.get(selectors.addModal.input).focus();
            cy.get(selectors.addModal.input).blur();

            cy.get(selectors.addModal.validationError).should('contain', 'required');
        });

        it('should show validation error when missing tag separator', () => {
            visitBaseImages();

            openAddModal();
            cy.get(selectors.addModal.input).type('ubuntu');
            cy.get(selectors.addModal.input).blur();

            cy.get(selectors.addModal.validationError).should(
                'contain',
                'must include both repository and tag'
            );
        });

        it('should enable save button with valid input', () => {
            visitBaseImages();

            openAddModal();
            cy.get(selectors.addModal.input).type('docker.io/library/ubuntu:22.04');

            cy.get(selectors.addModal.saveButton).should('not.be.disabled');
        });

        it('should add base image successfully', () => {
            const newBaseImage = 'docker.io/library/alpine:3.18';

            cy.intercept('POST', '/v2/baseimages', {
                statusCode: 200,
                body: {
                    id: '3',
                    baseImageRepoPath: 'docker.io/library/alpine',
                    baseImageTagPattern: '3.18',
                    user: { id: '1', username: 'admin', name: 'Admin User' },
                },
            }).as('addBaseImage');

            cy.intercept('GET', '/v2/baseimages', {
                statusCode: 200,
                body: {
                    baseImageReferences: [
                        {
                            id: '3',
                            baseImageRepoPath: 'docker.io/library/alpine',
                            baseImageTagPattern: '3.18',
                            user: { id: '1', username: 'admin', name: 'Admin User' },
                        },
                    ],
                },
            }).as('getBaseImages');

            visitBaseImages();

            openAddModal();
            cy.get(selectors.addModal.input).type(newBaseImage);
            cy.get(selectors.addModal.saveButton).click();

            cy.wait('@addBaseImage');
            cy.get(selectors.addModal.successAlert).should('be.visible');

            // Verify table shows new entry
            cy.get('td').should('contain', 'docker.io/library/alpine:3.18');
        });

        it('should show error alert when add fails', () => {
            cy.intercept('POST', '/v2/baseimages', {
                statusCode: 500,
                body: { message: 'Internal server error' },
            }).as('addBaseImageError');

            visitBaseImages();

            openAddModal();
            cy.get(selectors.addModal.input).type('docker.io/library/ubuntu:22.04');
            cy.get(selectors.addModal.saveButton).click();

            cy.wait('@addBaseImageError');
            cy.get(selectors.addModal.errorAlert).should('be.visible');
        });

        it('should close modal when clicking Cancel', () => {
            visitBaseImages();

            openAddModal();
            cy.get(selectors.addModal.cancelButton).click();

            cy.get(selectors.addModal.title).should('not.exist');
        });
    });

    describe('Delete base image', () => {
        const mockBaseImages = {
            baseImageReferences: [
                {
                    id: '1',
                    baseImageRepoPath: 'library/ubuntu',
                    baseImageTagPattern: '20.04.*',
                    user: { id: '1', username: 'admin', name: 'Admin User' },
                },
                {
                    id: '2',
                    baseImageRepoPath: 'library/alpine',
                    baseImageTagPattern: '3.*',
                    user: { id: '2', username: 'admin', name: 'Admin User' },
                },
            ],
        };

        it('should open delete confirmation modal', () => {
            cy.intercept('GET', '/v2/baseimages', {
                statusCode: 200,
                body: mockBaseImages,
            }).as('getBaseImages');

            visitBaseImages();

            cy.get('tr:contains("library/ubuntu") button[aria-label="Kebab toggle"]').click();
            cy.get(selectors.removeAction).click();

            cy.get(selectors.deleteModal.title).should('be.visible');
            cy.contains('library/ubuntu:20.04.*').should('be.visible');
        });

        it('should delete base image successfully', () => {
            cy.intercept('GET', '/v2/baseimages', {
                statusCode: 200,
                body: mockBaseImages,
            }).as('getBaseImages');

            cy.intercept('DELETE', '/v2/baseimages/1', {
                statusCode: 200,
                body: {},
            }).as('deleteBaseImage');

            // After deletion, return list without deleted item
            cy.intercept('GET', '/v2/baseimages', {
                statusCode: 200,
                body: {
                    baseImageReferences: [mockBaseImages.baseImageReferences[1]],
                },
            }).as('getBaseImagesAfterDelete');

            visitBaseImages();

            cy.get('tr:contains("library/ubuntu") button[aria-label="Kebab toggle"]').click();
            cy.get(selectors.removeAction).click();
            cy.get(selectors.deleteModal.confirmButton).click();

            cy.wait('@deleteBaseImage');

            // Verify deleted row is gone
            cy.get('td').should('not.contain', 'library/ubuntu:20.04.*');
            cy.get('td').should('contain', 'library/alpine:3.*');
        });

        it('should close modal when clicking Cancel', () => {
            cy.intercept('GET', '/v2/baseimages', {
                statusCode: 200,
                body: mockBaseImages,
            }).as('getBaseImages');

            visitBaseImages();

            cy.get('tr:contains("library/ubuntu") button[aria-label="Kebab toggle"]').click();
            cy.get(selectors.removeAction).click();
            cy.get(selectors.deleteModal.cancelButton).click();

            cy.get(selectors.deleteModal.title).should('not.exist');
            // Row should still be visible
            cy.get('td').should('contain', 'library/ubuntu:20.04.*');
        });

        it('should show error alert when delete fails', () => {
            cy.intercept('GET', '/v2/baseimages', {
                statusCode: 200,
                body: mockBaseImages,
            }).as('getBaseImages');

            cy.intercept('DELETE', '/v2/baseimages/1', {
                statusCode: 500,
                body: { message: 'Failed to delete' },
            }).as('deleteBaseImageError');

            visitBaseImages();

            cy.get('tr:contains("library/ubuntu") button[aria-label="Kebab toggle"]').click();
            cy.get(selectors.removeAction).click();
            cy.get(selectors.deleteModal.confirmButton).click();

            cy.wait('@deleteBaseImageError');
            cy.get(selectors.deleteModal.errorAlert).should('be.visible');
        });
    });
});
