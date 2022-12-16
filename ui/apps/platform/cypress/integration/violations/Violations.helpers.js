import { visitFromLeftNav } from '../../helpers/nav';
import { interactAndWaitForResponses } from '../../helpers/request';
import { visit } from '../../helpers/visit';

// Source of truth for keys in routeMatcherMap and staticResponseMap objects.
export const alertsAlias = 'alerts';
export const alertsCountAlias = 'alertscount';

const routeMatcherMapForViolationsWithoutSearchOptions = {
    [alertsAlias]: {
        method: 'GET',
        url: '/v1/alerts?query=*',
    },
    [alertsCountAlias]: {
        method: 'GET',
        url: '/v1/alertscount?query=*',
    },
};

const searchOptionsAlias = 'search/metadata/options';

const routeMatcherMapForSearchOptions = {
    [searchOptionsAlias]: {
        method: 'GET',
        url: '/v1/search/metadata/options?categories=ALERTS',
    },
};

const routeMatcherMapForViolationsWithSearchOptions = {
    ...routeMatcherMapForViolationsWithoutSearchOptions,
    ...routeMatcherMapForSearchOptions,
};

const basePath = '/main/violations';

const title = 'Violations';

// visit

export function visitViolationsFromLeftNav() {
    visitFromLeftNav(title, routeMatcherMapForViolationsWithSearchOptions);

    cy.location('pathname').should('eq', basePath);
    cy.get(`h1:contains("${title}")`);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitViolations(staticResponseMap) {
    visit(basePath, routeMatcherMapForViolationsWithSearchOptions, staticResponseMap);

    cy.get(`.pf-c-page__sidebar nav.pf-c-nav > ul > li > a:contains("${title}")`).should(
        'have.class',
        'pf-m-current'
    );
    cy.get(`h1:contains("${title}")`);
}

export function visitViolationsWithFixture(fixturePath) {
    cy.fixture(fixturePath).then(({ alerts }) => {
        const count = alerts.length;
        const staticResponseMap = {
            [alertsAlias]: { body: { alerts } },
            [alertsCountAlias]: { body: { count } },
        };

        visit(basePath, routeMatcherMapForViolationsWithSearchOptions, staticResponseMap);

        cy.get(`h1:contains("${title}")`);
    });
}

const alertAlias = 'alerts/id';

/*
 * Assume that current location is violations table with compatible fixture for alerts.
 */
export function visitViolationFromTableWithFixture(fixturePath) {
    cy.fixture(fixturePath).then((alert) => {
        const { id, policy } = alert;
        const { name } = policy;

        const routeMatcherMapForViolation = {
            [alertAlias]: {
                method: 'GET',
                url: `/v1/alerts/${id}`,
            },
        };

        const staticResponseMapForViolation = {
            [alertAlias]: {
                body: alert,
            },
        };

        interactAndWaitForResponses(
            () => {
                // Make sure the policy name matches only one row in the table.
                cy.get(`td[data-label="Policy"] a:contains("${name}")`).click();
            },
            routeMatcherMapForViolation,
            staticResponseMapForViolation
        );

        cy.get(`h1:contains("${name}")`);
    });
}

/*
 * Visit violation page directly.
 */
export function visitViolationWithFixture(fixturePath) {
    cy.fixture(fixturePath).then((alert) => {
        const { id, policy } = alert;
        const { name } = policy;

        const routeMatcherMapForViolation = {
            [alertAlias]: {
                method: 'GET',
                url: `/v1/alerts/${id}`,
            },
        };

        const staticResponseMapForViolation = {
            [alertAlias]: {
                body: alert,
            },
        };

        visit(`${basePath}/${id}`, routeMatcherMapForViolation, staticResponseMapForViolation);

        cy.get(`h1:contains("${name}")`);
    });
}

// interact

/*
 * Assume that current location is violations table without fixture.
 */
export function interactAndWaitForViolationsResponses(interactionCallback) {
    interactAndWaitForResponses(
        interactionCallback,
        routeMatcherMapForViolationsWithoutSearchOptions
    );
}

/*
 * Assume that current location is violation page with compatible fixture for alert.
 */
export function clickDeploymentTabWithFixture(fixturePath) {
    const deploymentAlias = 'deployments/id';

    const routeMatcherMapForDeployment = {
        [deploymentAlias]: {
            method: 'GET',
            url: '/v1/deployments/*',
        },
    };

    const staticResponseMapForDeployment = {
        [deploymentAlias]: {
            fixture: fixturePath,
        },
    };

    const deploymentTab = 'li.pf-c-tabs__item:contains("Deployment")';

    cy.get(deploymentTab).should('not.have.class', 'pf-m-current');

    interactAndWaitForResponses(
        () => {
            cy.get(deploymentTab).click();
        },
        routeMatcherMapForDeployment,
        staticResponseMapForDeployment
    );

    cy.get(deploymentTab).should('have.class', 'pf-m-current');
}
