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
 * @returns {{ request: Record<string, unknown>, response: Record<string, unknown>}[]}
 */
export function waitForResponses(routeMatcherMap, waitOptions = {}) {
    if (routeMatcherMap) {
        const aliases = Object.keys(routeMatcherMap).map((alias) => `@${alias}`);

        return cy.wait(aliases, waitOptions);
    }

    return [];
}

/**
 * Intercept requests before interaction and then wait for responses.
 *
 * @param {() => void} interactionCallback
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMap]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 * @param {Parameters<Cypress.Chainable['wait']>[1]} [waitOptions]

 * @returns {{ request: Record<string, unknown>, response: Record<string, unknown>}[]}
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
