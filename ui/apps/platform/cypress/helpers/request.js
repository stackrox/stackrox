/*
 * For pages which have GraphQL and REST requests.
 *
 * For example, given 'searchOptions' return:
 * {
 *     method: 'POST',
 *     url: '/api/graphql?opname=searchOptions',
 * }
 */
export function getRouteMatcherForGraphQL(opname) {
    return {
        method: 'POST',
        url: `/api/graphql?opname=${opname}`,
    };
}

/*
 * For pages which have only GraphQL requests.
 *
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
 */
export function getRouteMatcherMapForGraphQL(opnames) {
    const routeMatcherMap = {};

    opnames.forEach((opname) => {
        routeMatcherMap[opname] = getRouteMatcherForGraphQL(opname);
    });

    return routeMatcherMap;
}

/*
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

/*
 * Wait for responses after initial page visit or subsequent interaction.
 *
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMap]
 */
export function waitForResponses(routeMatcherMap) {
    if (routeMatcherMap) {
        const aliases = Object.keys(routeMatcherMap).map((alias) => `@${alias}`);

        cy.wait(aliases);
    }
}

/*
 * Intercept requests before interaction and then wait for responses.
 *
 * @param {() => void} interactionCallback
 * @param {Record<string, { method: string, url: string }>} [routeMatcherMap]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function interactAndWaitForResponses(
    interactionCallback,
    routeMatcherMap,
    staticResponseMap
) {
    interceptRequests(routeMatcherMap, staticResponseMap);

    interactionCallback();

    waitForResponses(routeMatcherMap);
}
