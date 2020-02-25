/*
 * Adds mockGraphQL cypress command that uses the operationName of GraphQL queries
 * in order to stub GraphQL APIs. Implementation learned from the following medium post
 *
 * https://medium.com/wehkamp-techblog/mocking-specific-graphql-requests-in-cypress-io-22c67924e296
 */

const responseStub = result =>
    Promise.resolve({
        json() {
            return Promise.resolve(result);
        },
        text() {
            return Promise.resolve(JSON.stringify(result));
        },
        ok: result.ok === undefined ? true : result.ok
    });

function isStub(wrappedMethod) {
    return wrappedMethod.restore && wrappedMethod.restore.sinon;
}

let originalFetch;

function getFetchStub(win, { withArgs, as, callsFake }) {
    let stub = win.fetch;
    if (!isStub(win.fetch)) {
        originalFetch = win.fetch;
        stub = cy.stub(win, 'fetch');
    }
    stub.withArgs(...withArgs)
        .as(as)
        .callsFake(callsFake);
}

Cypress.Commands.add('mockGraphQL', getOperationMock => {
    const fetchGraphQL = (path, options, ...rest) => {
        const { body } = options;
        try {
            const { operationName, variables } = JSON.parse(body);
            const { test = null, mockResult } = getOperationMock(operationName);

            if (typeof test === 'function') {
                test(variables);
            }
            if (mockResult) {
                return responseStub(mockResult);
            }
            return originalFetch(path, options, ...rest);
        } catch (err) {
            return responseStub(err);
        }
    };

    return cy.on('window:before:load', win => {
        getFetchStub(win, {
            withArgs: ['/api/graphql'],
            as: 'fetchGraphQL',
            callsFake: fetchGraphQL
        });
    });
});
