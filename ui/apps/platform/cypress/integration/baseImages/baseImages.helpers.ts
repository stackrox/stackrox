import { interactAndWaitForResponses } from '../../helpers/request';
import { visitFromLeftNavExpandable } from '../../helpers/nav';
import { visit } from '../../helpers/visit';

const basePath = '/main/base-images';
const pageTitle = 'h1:contains("Base Images")';

const routeMatcherMapForGET = {
    baseImages: {
        method: 'GET',
        url: '/v2/baseimages',
    },
};

export function visitBaseImages(
    staticResponseMap?: Record<string, { body: unknown } | { fixture: string }>
) {
    visit(basePath, routeMatcherMapForGET, staticResponseMap);
    cy.get(pageTitle);
}

export function visitBaseImagesFromLeftNav() {
    visitFromLeftNavExpandable('Platform Configuration', 'Base Images', routeMatcherMapForGET, {});
    cy.location('pathname').should('eq', basePath);
    cy.get(pageTitle);
}

export function openAddModal() {
    cy.get('button:contains("Add base image")').click();
    cy.get('h2:contains("Add base image path")').should('be.visible');
}

export function addBaseImage(imagePath: string) {
    openAddModal();
    interactAndWaitForResponses(
        () => {
            cy.get('input#baseImagePath').type(imagePath);
            cy.get('button:contains("Save")').click();
        },
        {
            addBaseImage: { method: 'POST', url: '/v2/baseimages' },
            baseImages: { method: 'GET', url: '/v2/baseimages' },
        }
    );

    // Wait for modal to close before proceeding
    cy.get('[role="dialog"]').should('not.exist');
}
