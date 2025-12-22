import { visitFromLeftNavExpandable } from '../../helpers/nav';
import { interactAndWaitForResponses } from '../../helpers/request';
import { visit } from '../../helpers/visit';

import { selectors } from './baseImages.selectors';

// Route configuration
export const basePath = '/main/base-images';

const baseImagesEndpoint = '/v2/baseimages';

export const baseImagesAlias = 'baseImages';

export const routeMatcherMapForGET = {
    [baseImagesAlias]: {
        method: 'GET',
        url: baseImagesEndpoint,
    },
};

const routeMatcherMapForPOST = {
    addBaseImage: {
        method: 'POST',
        url: baseImagesEndpoint,
    },
};

const routeMatcherMapForDELETE = {
    deleteBaseImage: {
        method: 'DELETE',
        url: `${baseImagesEndpoint}/*`,
    },
};

// Visit functions
export function visitBaseImages(
    staticResponseMap?: Record<string, { body: unknown } | { fixture: string }>
) {
    visit(basePath, routeMatcherMapForGET, staticResponseMap);
    cy.get(selectors.pageTitle);
}

export function visitBaseImagesFromLeftNav() {
    visitFromLeftNavExpandable('Platform Configuration', 'Base Images', routeMatcherMapForGET, {});
    cy.location('pathname').should('eq', basePath);
    cy.get(selectors.pageTitle);
}

// Interaction functions
export function openAddModal() {
    cy.get(selectors.addButton).click();
    cy.get(selectors.addModal.title).should('be.visible');
}

export function addBaseImage(
    baseImagePath: string,
    staticResponseMap?: Record<string, { body: unknown } | { fixture: string }>
) {
    openAddModal();
    cy.get(selectors.addModal.input).type(baseImagePath);
    interactAndWaitForResponses(
        () => {
            cy.get(selectors.addModal.saveButton).click();
        },
        { ...routeMatcherMapForPOST, ...routeMatcherMapForGET },
        staticResponseMap
    );
}

export function deleteBaseImage(
    baseImagePath: string,
    staticResponseMap?: Record<string, { body: unknown } | { fixture: string }>
) {
    // Find the row with the base image and click the kebab menu
    cy.get(`tr:contains("${baseImagePath}") ${selectors.rowKebabButton}`).click();
    cy.get(selectors.removeAction).click();

    // Confirm deletion
    cy.get(selectors.deleteModal.title).should('be.visible');
    interactAndWaitForResponses(
        () => {
            cy.get(selectors.deleteModal.confirmButton).click();
        },
        { ...routeMatcherMapForDELETE, ...routeMatcherMapForGET },
        staticResponseMap
    );
}

export function cancelDeleteBaseImage() {
    cy.get(selectors.deleteModal.cancelButton).click();
    cy.get(selectors.deleteModal.title).should('not.exist');
}
