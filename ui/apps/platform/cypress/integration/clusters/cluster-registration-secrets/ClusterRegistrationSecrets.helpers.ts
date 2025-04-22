export function cleanupClusterRegistrationSecretsWithName(nameToDelete: string) {
    // Clean up existing CRSs, if they exist
    const auth = { bearer: Cypress.env('ROX_AUTH_TOKEN') };

    cy.request({ url: '/v1/cluster-init/crs', auth }).as('listCrs');

    return cy.get('@listCrs').then((res: any) => {
        const automationTokens = res.body.items.filter(({ name }) => name === nameToDelete);
        const body = { ids: automationTokens.map(({ id }) => id) };
        return cy.request({ url: '/v1/cluster-init/crs/revoke', body, auth, method: 'PATCH' });
    });
}
