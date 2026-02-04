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
    cy.intercept('POST', '/v2/baseimages').as('addBaseImage');
    openAddModal();
    cy.get('input#baseImagePath').type(imagePath);
    cy.get('button:contains("Save")').click();
    cy.wait('@addBaseImage');

    // Close the modal after adding
    cy.get('[role="dialog"] button[aria-label="Close"]').click();
}
