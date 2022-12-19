import { selectors as topNavSelectors } from '../../constants/TopNavigation';
import { visitFromLeftNavExpandable } from '../../helpers/nav';
import { interactAndWaitForResponses } from '../../helpers/request';
import { visit, visitWithStaticResponseForPermissions } from '../../helpers/visit';

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

// interact

export function logOut() {
    cy.get(topNavSelectors.menuButton).click();
    cy.get(topNavSelectors.menuList.logoutButton).click();
}
