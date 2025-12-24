import { visitFromLeftNavExpandable } from '../../helpers/nav';
import { visit } from '../../helpers/visit';

import { selectors } from './baseImages.selectors';

const basePath = '/main/base-images';

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
    cy.get(selectors.pageTitle);
}

export function visitBaseImagesFromLeftNav() {
    visitFromLeftNavExpandable('Platform Configuration', 'Base Images', routeMatcherMapForGET, {});
    cy.location('pathname').should('eq', basePath);
    cy.get(selectors.pageTitle);
}

export function openAddModal() {
    cy.get(selectors.addButton).click();
    cy.get(selectors.addModal.title).should('be.visible');
}
