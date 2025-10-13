import { interactAndWaitForResponses } from '../../helpers/request';

import { routeMatcherMapWithoutStandards } from './Compliance.helpers';

// selectors

export const selectorForModal = `[role="dialog"]:has('h1:contains("Manage standards")')`;

export function selectorInModal(selector) {
    return `${selectorForModal} ${selector}`;
}
export function selectorForWidget(title) {
    return `.widget:has('[data-testid="widget-header"]:contains("${title}")')`;
}

export function selectorInWidget(title, selector) {
    return `${selectorForWidget(title)} ${selector}`;
}

// interactions

export function openModal() {
    const routeMatcherMap = {
        'compliance/standards': {
            method: 'GET',
            url: '/v1/compliance/standards',
        },
    };

    interactAndWaitForResponses(() => {
        cy.get('button:contains("Manage standards")').click();
    }, routeMatcherMap);

    cy.get(selectorForModal);
}

export function clickSaveAndWaitForPatchComplianceStandards(standardIds) {
    const routeMatcherMapForPatch = Object.fromEntries(
        standardIds.map((standardId) => [
            `PATCH_${standardId}`,
            { method: 'PATCH', url: `/v1/compliance/standards/${standardId}` },
        ])
    );
    const routeMatcherMap = {
        ...routeMatcherMapForPatch,
        ...routeMatcherMapWithoutStandards,
    };

    return interactAndWaitForResponses(() => {
        cy.get(selectorInModal('button:contains("Save")')).click();
    }, routeMatcherMap); // TODO GraphQL requests?
}
