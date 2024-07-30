/**
 * For example, given ['searchOptions', 'getDeployments'] return:
 * {
 *     searchOptions: {
 *         method: 'POST',
 *         url: '/api/graphql?opname=searchOptions',
 *     },
 *     getDeployments: {
 *         method: 'POST',
 *         url: '/api/graphql?opname=getDeployments',
 *     },
 * }
 *
 * Remember to enclose single opname in array brackets. For example, ['searchOptions']
 *
 * Use object spread to merge GraphQL object into object which has properties for REST requests.
 *
 * @param {[string]} opnames
 * @returns Record<string, { method: string, url: string }>
 */
export function getRouteMatcherMapForGraphQL(opnames) {
    /** @type Record<string, { method: string, url: string }> */
    const routeMatcherMap = {};

    opnames.forEach((opname) => {
        routeMatcherMap[opname] = {
            method: 'POST',
            url: `/api/graphql?opname=${opname}`,
        };
    });

    return routeMatcherMap;
}

/**
 * Intercept requests before initial page visit or subsequent interaction:
 * routeMatcherMap: { alias: routeMatcher, … }
 *
 * Optionally replace responses with stub for routeMatcher alias key:
 * staticResponseMap: { alias: { body }, … }
 * staticResponseMap: { alias: { fixture }, … }
 *
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMap]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function interceptRequests(routeMatcherMap, staticResponseMap) {
    if (routeMatcherMap) {
        Object.entries(routeMatcherMap).forEach(([key, routeMatcher]) => {
            if (staticResponseMap?.[key]) {
                const staticResponse = staticResponseMap[key];
                cy.intercept(routeMatcher, staticResponse).as(key);
            } else {
                cy.intercept(routeMatcher).as(key);
            }
        });
    }
}

/**
 * Wait for responses after initial page visit or subsequent interaction.
 *
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMap]
 * @param {Parameters<Cypress.Chainable['wait']>[1]} [waitOptions]
 * @returns {Cypress.Chainable<Interception[] | Interception>}
 */
export function waitForResponses(routeMatcherMap, waitOptions = {}) {
    if (routeMatcherMap) {
        const aliases = Object.keys(routeMatcherMap).map((alias) => `@${alias}`);

        return cy.wait(aliases, waitOptions);
    }

    return cy.wrap([]);
}

/**
 * Intercept requests before interaction and then wait for responses.
 *
 * @param {() => void} interactionCallback
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMap]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 * @param {Parameters<Cypress.Chainable['wait']>[1]} [waitOptions]
 * @returns {Cypress.Chainable<Interception[] | Interception>}
 */
export function interactAndWaitForResponses(
    interactionCallback,
    routeMatcherMap,
    staticResponseMap,
    waitOptions
) {
    interceptRequests(routeMatcherMap, staticResponseMap);

    interactionCallback();

    return waitForResponses(routeMatcherMap, waitOptions);
}

/**
 * Intercepts a GraphQL request during an interaction and yields the
 * variables object passed to the request's query
 *
 * @param {() => void} interactionCallback The interaction performed to trigger the GraphQL request
 * @param {string} opname The GraphQL operation name
 * @returns {Cypress.Chainable<Interception>}
 */
export function interactAndInspectGraphQLVariables(interactionCallback, opname) {
    const url = `/api/graphql?opname=${opname}`;

    cy.intercept({ method: 'POST', url, times: 1 }).as(opname);

    interactionCallback();

    return cy.wait(`@${opname}`).then((interception) => {
        return cy.wrap(interception.request.body.variables);
    });
}

export function expectRequestedSort(expectedSort) {
    return (variables) => {
        const { sortOption } = variables.pagination;
        expect(sortOption).to.deep.equal(
            expectedSort,
            `Expected sort option ${JSON.stringify(expectedSort)} but received ${JSON.stringify(sortOption)}`
        );
    };
}
