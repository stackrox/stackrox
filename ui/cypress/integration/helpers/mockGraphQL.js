const mockGraphQL = (operationName, mockResult) => {
    cy.mockGraphQL(gqlOperationName => {
        switch (gqlOperationName) {
            case operationName:
                return {
                    mockResult
                };
            default:
                return {};
        }
    });
};

export default mockGraphQL;
