import { visit, visitWithStaticResponseForPermissions } from '../../helpers/visit';

export const vulnerabilitiesConfigAlias = 'config/private/deferral/vulnerabilities';

const routeMatcherMapForVulnerabilitiesExceptionConfig = {
    [vulnerabilitiesConfigAlias]: {
        method: 'GET',
        url: '/v1/config/private/exception/vulnerabilities',
    },
};

const basePath = '/main/exception-configuration';

export function visitExceptionConfig(category, staticResponseMap) {
    const path = category ? `${basePath}?category=${category}` : basePath;
    visit(path, routeMatcherMapForVulnerabilitiesExceptionConfig, staticResponseMap);
}

export function visitExceptionConfigWithPermissions(category, resourceToAccess) {
    const path = category ? `${basePath}?category=${category}` : basePath;
    visitWithStaticResponseForPermissions(path, {
        body: { resourceToAccess },
    });
}

/**
 * Sets the exception config to the expected default as defined in the requirements document.
 */
export function resetExceptionConfig() {
    const auth = { bearer: Cypress.env('ROX_AUTH_TOKEN') };
    const config = {
        expiryOptions: {
            dayOptions: [
                { numDays: 14, enabled: true },
                { numDays: 30, enabled: true },
                { numDays: 90, enabled: true },
            ],
            fixableCveOptions: {
                allFixable: true,
                anyFixable: true,
            },
            customDate: false,
            indefinite: false,
        },
    };

    cy.request({
        url: `/v1/config/private/exception/vulnerabilities`,
        auth,
        method: 'PUT',
        body: { config },
    });
}
