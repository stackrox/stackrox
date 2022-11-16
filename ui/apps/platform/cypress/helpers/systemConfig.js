import { visitFromLeftNavExpandable } from './nav';
import { interactAndWaitForResponses } from './request';
import { visit, visitWithStaticResponseForPermissions } from './visit';

const basePath = '/main/systemconfig';

const configEndpoint = '/v1/config';

const configAliasForGET = 'config';

const routeMatcherMapForGET = {
    [configAliasForGET]: {
        method: 'GET',
        url: configEndpoint,
    },
};

const title = 'System Configuration';

// visit

export function visitSystemConfiguration() {
    visit(basePath, routeMatcherMapForGET);

    cy.get(`h1:contains("${title}")`);
}

export function visitSystemConfigurationFromLeftNav() {
    visitFromLeftNavExpandable('Platform Configuration', title, routeMatcherMapForGET);

    cy.location('pathname').should('eq', basePath);
    cy.get(`h1:contains("${title}")`);
}

export function visitSystemConfigurationWithStaticResponseForPermissions(
    staticResponseForPermissions
) {
    visitWithStaticResponseForPermissions(
        basePath,
        staticResponseForPermissions,
        routeMatcherMapForGET
    );

    cy.get(`h1:contains("${title}")`);
}

// save

const configAliasForPUT = 'PUT_config';

const routeMatcherMapForPUT = {
    [configAliasForPUT]: {
        method: 'PUT',
        url: configEndpoint,
    },
};

export function saveSystemConfiguration() {
    interactAndWaitForResponses(() => {
        cy.get('button:contains("Save")').click();
    }, routeMatcherMapForPUT);
}
