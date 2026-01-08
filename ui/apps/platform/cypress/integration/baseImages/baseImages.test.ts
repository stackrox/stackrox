import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import { addBaseImage, visitBaseImages, visitBaseImagesFromLeftNav } from './baseImages.helpers';

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
        cy.get('h1:contains("Base Images")');
        cy.get('p:contains("Manage approved base images")');

        // Verify table renders with expected headers
        cy.get('table');
        cy.get('th:contains("Base image path")');
        cy.get('th:contains("Added by")');
    });

    it('should add base image and update table', () => {
        const newBaseImage = 'docker.io/library/alpine:3.18';

        visitBaseImages();
        addBaseImage(newBaseImage);

        // Verify table shows new entry
        cy.get('table tbody tr').should('contain', newBaseImage);
    });

    it('should delete base image successfully', () => {
        const testBaseImage = 'docker.io/library/nginx:1.25';

        visitBaseImages();
        addBaseImage(testBaseImage);

        // Set up DELETE intercept
        cy.intercept('DELETE', '/v2/baseimages/*').as('deleteBaseImage');

        // Delete the base image
        cy.get('table tbody tr').contains('td', testBaseImage).parents('tr').as('targetRow');
        cy.get('@targetRow').find('button[aria-label="Kebab toggle"]').click();
        cy.get('button:contains("Delete base image")').click();
        cy.get('[role="dialog"] button:contains("Delete")').click();

        // Wait for delete to complete
        cy.wait('@deleteBaseImage');

        // Verify it's gone
        cy.get('table tbody tr').should('not.contain', testBaseImage);
    });
});
