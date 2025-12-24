import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import { openAddModal, visitBaseImages, visitBaseImagesFromLeftNav } from './baseImages.helpers';

describe('Base Images', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_BASE_IMAGE_DETECTION')) {
            this.skip();
        }
    });

    it('should navigate to Base Images page from left nav', () => {
        visitBaseImagesFromLeftNav();

        // Verify page loaded correctly
        cy.title().should('match', getRegExpForTitleWithBranding('Base Images'));
        cy.get('h1:contains("Base Images")').should('be.visible');
        cy.get('p:contains("Manage approved base images")').should('be.visible');

        // Verify table renders with expected headers
        cy.get('table').should('exist');
        cy.get('th:contains("Base image path")').should('be.visible');
        cy.get('th:contains("Added by")').should('be.visible');
    });

    it('should add base image and update table', () => {
        const newBaseImage = 'docker.io/library/alpine:3.18';

        visitBaseImages();

        openAddModal();
        cy.get('input#baseImagePath').type(newBaseImage);
        cy.get('button:contains("Save")').click();

        // Verify table shows new entry
        cy.get('td').should('contain', 'docker.io/library/alpine:3.18');
    });

    it('should delete base image successfully', () => {
        visitBaseImages();

        // Get initial row count
        cy.get('table tbody tr').then(($rows) => {
            const initialCount = $rows.length;

            // Click first row's kebab menu
            cy.get('table tbody tr').first().find('button[aria-label="Kebab toggle"]').click();
            cy.get('button:contains("Remove")').click();
            cy.get('*[role="dialog"] button:contains("Delete")').click();

            // Verify row count decreased
            cy.get('table tbody tr').should('have.length', initialCount - 1);
        });
    });
});
