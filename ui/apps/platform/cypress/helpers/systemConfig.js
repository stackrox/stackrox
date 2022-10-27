import { visitFromLeftNavExpandable } from './nav';
import { interactAndWaitForResponses } from './request';
import { visit, visitWithStaticResponseForPermissions } from './visit';

const basePath = '/main/systemconfig';

const configEndpoint = '/v1/config';

const configAliasForGET = 'config';

const requestConfigForGET = {
    routeMatcherMap: {
        [configAliasForGET]: {
            method: 'GET',
            url: configEndpoint,
        },
    },
};

const title = 'System Configuration';

// visit

export function visitSystemConfiguration() {
    visit(basePath, requestConfigForGET);

    cy.get(`h1:contains("${title}")`);
}

export function visitSystemConfigurationFromLeftNav() {
    visitFromLeftNavExpandable('Platform Configuration', title, requestConfigForGET);

    cy.get(`h1:contains("${title}")`);
    cy.location('pathname').should('eq', basePath);
}

export function visitSystemConfigurationWithStaticResponseForPermissions(
    staticResponseForPermissions
) {
    visitWithStaticResponseForPermissions(
        basePath,
        staticResponseForPermissions,
        requestConfigForGET
    );

    cy.get(`h1:contains("${title}")`);
}

// save

const configAliasForPUT = 'PUT_config';

const requestConfigForPUT = {
    routeMatcherMap: {
        [configAliasForPUT]: {
            method: 'PUT',
            url: configEndpoint,
        },
    },
};

export function saveSystemConfiguration() {
    interactAndWaitForResponses(() => {
        cy.get('button:contains("Save")').click();
    }, requestConfigForPUT);
}
