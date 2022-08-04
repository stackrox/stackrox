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
 */
export function interactAndWaitForResponses(interactionCallback, requestConfig, staticResponseMap) {
    interceptRequests(requestConfig, staticResponseMap);

    interactionCallback();

    waitForResponses(requestConfig);
}
