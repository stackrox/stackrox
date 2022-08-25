/*
 * Intercept requests before initial page visit or subsequent interaction:
 * routeMatcherMap: { key: routeMatcher, … }
 *
 * Optionally replace responses with stub for routeMatcher alias key:
 * staticResponseMap: { alias: { body }, … }
 * staticResponseMap: { alias: { fixture }, … }
 *
 * Optionally assign aliases for multiple GraphQL requests with routeMatcher opname key:
 * opnameAliasesMap: { opname: { aliases, routeHandler }, … }
 *
 * @param {{ routeMatcherMap?: Record<string, { method: string, url: string }>, opnameAliasesMap?: Record<string, (request: Object) => boolean> }} [requestConfig]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function interceptRequests(requestConfig, staticResponseMap) {
    if (requestConfig?.routeMatcherMap) {
        const { opnameAliasesMap, routeMatcherMap } = requestConfig;

        Object.entries(routeMatcherMap).forEach(([key, routeMatcher]) => {
            if (opnameAliasesMap?.[key]) {
                const aliasesMap = opnameAliasesMap[key];
                const routeHandler = (req) => {
                    const aliasFound = Object.keys(aliasesMap).find((alias) => {
                        const aliasReqPredicate = aliasesMap[alias];
                        return aliasReqPredicate(req);
                    });
                    if (typeof aliasFound === 'string') {
                        req.alias = aliasFound;
                    }
                };
                cy.intercept(routeMatcher, routeHandler);
            } else if (staticResponseMap?.[key]) {
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
 * Optionally wait with waitOptions: { requestTimeout, responseTimeout }
 *
 * @param {{ routeMatcherMap?: Record<string, { method: string, url: string }>, opnameAliasesMap?: Record<string, (request: Object) => boolean>, waitOptions?: { requestTimeout?: number, responseTimeout?: number } }} [requestConfig]
 */
export function waitForResponses(requestConfig) {
    if (requestConfig?.routeMatcherMap) {
        const { opnameAliasesMap, routeMatcherMap, waitOptions } = requestConfig;

        const aliases = Object.keys(routeMatcherMap)
            .map((key) => {
                const aliasesMap = opnameAliasesMap?.[key];
                if (aliasesMap) {
                    return Object.keys(aliasesMap);
                }
                return key;
            })
            .flat()
            .map((alias) => `@${alias}`);

        cy.wait(aliases, waitOptions);
    }
}

/*
 * Intercept requests before interaction and then wait for responses.
 *
 * @param {() => void} interactionCallback
 * @param {{ routeMatcherMap?: Record<string, { method: string, url: string }>, opnameAliasesMap?: Record<string, (request: Object) => boolean>, waitOptions?: { requestTimeout?: number, responseTimeout?: number } }} [requestConfig]
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function interactAndWaitForResponses(interactionCallback, requestConfig, staticResponseMap) {
    interceptRequests(requestConfig, staticResponseMap);

    interactionCallback();

    waitForResponses(requestConfig);
}
