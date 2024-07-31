/**
 * @typedef {import("cypress/types/net-stubbing").RouteMatcherOptions} RouteMatcherOptions
 * @typedef {import("cypress/types/net-stubbing").RouteHandler} RouteHandler
 * @typedef {import("cypress/types/net-stubbing").WaitOptions} WaitOptions
 */

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
 * Intercept requests and monitor requests/responses across multiple interactions
 *
 * @template {string} RouteKey
 * @param {Record<RouteKey, RouteMatcherOptions>} routeMatcherMap
 * @param {Partial<Record<RouteKey, RouteHandler>>} [staticResponseMap]
 * @returns {Promise<{
 *    waitForRequests: typeof waitForRequests,
 *    waitAndYieldRequestBodyVariables: typeof waitAndYieldRequestBodyVariables,
 * }>} Helper functions used to monitor requests
 *          after an interaction that causes a request
 */
export function interceptAndWatchRequests(routeMatcherMap, staticResponseMap) {
    interceptRequests(routeMatcherMap, staticResponseMap);

    /**
     * Wait for requests to complete after an interaction
     *
     * @param {RouteKey[]=} keys The keys of the routeMatcherMap to wait for, if not provided, wait for all keys in routeMatcherMap
     * @param {WaitOptions=} waitOptions Wait options for cy.wait
     * @returns {Cypress.Chainable<Interception> | Cypress.Chainable<Interception[]>} The interception object or array of interception objects
     */
    function waitForRequests(keys, waitOptions) {
        const aliases =
            keys && keys.length > 0
                ? keys.map((key) => `@${key}`)
                : Object.keys(routeMatcherMap).map((key) => `@${key}`);

        return cy.wait(aliases, waitOptions);
    }

    /**
     * Wait for requests to complete after an interaction and yield the variables object passed in the request body
     *
     * @param {RouteKey[]=} keys The keys of the routeMatcherMap to wait for, if not provided, wait for all keys in routeMatcherMap
     * @param {WaitOptions=} waitOptions Wait options for cy.wait
     * @returns {Cypress.Chainable<any> | Cypress.Chainable<any[]>} The request variables object or array of request variables objects that were passed in the
     *          request body of the intercepted request, if available
     */
    function waitAndYieldRequestBodyVariables(keys, waitOptions) {
        return waitForRequests(keys, waitOptions).then((interception) => {
            if (Array.isArray(interception)) {
                return cy.wrap(interception.map(({ request }) => request.body.variables));
            }
            return cy.wrap(interception.request.body.variables);
        });
    }

    return cy.wrap({
        waitForRequests,
        waitAndYieldRequestBodyVariables,
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

export function expectRequestedQuery(expectedQuery) {
    return ({ query }) => {
        expect(query).to.deep.equal(
            expectedQuery,
            `Expected query ${expectedQuery} but received ${query}`
        );
    };
}

export function expectRequestedPagination(expectedPagination) {
    return (variables) => {
        const { pagination } = variables;
        delete pagination.sortOption;
        expect(pagination).to.deep.equal(
            expectedPagination,
            `Expected pagination ${JSON.stringify(expectedPagination)} but received ${JSON.stringify(pagination)}`
        );
    };
}
