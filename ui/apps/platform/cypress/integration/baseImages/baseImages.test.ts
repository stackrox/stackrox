import withAuth from '../../helpers/basicAuth';
import { interactAndWaitForResponses } from '../../helpers/request';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import { addBaseImage, visitBaseImages, visitBaseImagesFromLeftNav } from './baseImages.helpers';

describe('Base Images', () => {
    withAuth();

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

        // Delete the base image and wait for both the DELETE and the subsequent list refetch
        cy.get(`tr:has(td:contains("${testBaseImage}")) button[aria-label="Kebab toggle"]`).click();
        interactAndWaitForResponses(
            () => {
                cy.get('button:contains("Delete base image")').click();
                cy.get('[role="dialog"] button:contains("Delete")').click();
            },
            {
                deleteBaseImage: { method: 'DELETE', url: '/v2/baseimages/*' },
                baseImages: { method: 'GET', url: '/v2/baseimages' },
            }
        );

        // Verify the deleted image is no longer in the table
        cy.get(`tr:has(td:contains("${testBaseImage}"))`).should('not.exist');
    });
});
