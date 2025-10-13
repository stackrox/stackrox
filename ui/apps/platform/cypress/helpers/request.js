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
 * @param {string[]} opnames
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

export function toAliases(keys) {
    return keys.map((key) => `@${key}`);
}

/**
 * Given an object with keys that are aliases, return an array of @-prefixed aliases
 *
 * @param {Record<string, RouteMatcherOptions>} routeMatcherMap
 * @returns {string[]} An array of @-prefixed aliases
 */
export function aliasesFromRouteMatcher(routeMatcherMap) {
    return toAliases(Object.keys(routeMatcherMap));
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
        return cy.wait(aliasesFromRouteMatcher(routeMatcherMap), waitOptions);
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
            keys && keys.length > 0 ? toAliases(keys) : aliasesFromRouteMatcher(routeMatcherMap);

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

/**
 * Shorthand function to override user permissions. Will overwrite the permissions for the user
 * with the provided permissions, keeping the rest of the permissions the same.
 * @param {Record<string, string>} overrides Resource to access level mapping
 */
export function interceptAndOverridePermissions(overrides) {
    return cy.intercept('GET', '/v1/mypermissions', (req) => {
        req.continue((res) => {
            Object.entries(overrides).forEach(([resource, accessLevel]) => {
                res.body.resourceToAccess[resource] = accessLevel;
            });
        });
    });
}

/**
 * Shortcut function to override feature flags. Will overwrite the feature flags for the user
 * with the provided enabled/disabled statuses, keeping the rest of the feature flags the same.
 * @param {Record<string, boolean>} overrides Feature flag env var to enabled boolean mapping
 */
export function interceptAndOverrideFeatureFlags(overrides) {
    return cy.intercept('GET', '/v1/featureflags', (req) =>
        req.continue((res) => {
            Object.entries(overrides).forEach(([feature, enabled]) => {
                const flag = res.body.featureFlags.find((flag) => flag.envVar === feature);
                if (flag) {
                    flag.enabled = enabled;
                }
            });
        })
    );
}

export function expectRequestedSort(expectedSort) {
    return (variables) => {
        const { sortOption, sortOptions } = variables.pagination;
        const targetSortOption = typeof sortOptions === 'undefined' ? sortOption : sortOptions;
        expect(targetSortOption).to.deep.equal(
            expectedSort,
            `Expected sort option ${JSON.stringify(expectedSort)} but received ${JSON.stringify(targetSortOption)}`
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
