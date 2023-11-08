function debugUrl(url) {
    const auth = { bearer: Cypress.env('ROX_AUTH_TOKEN') };

    cy.log(url);
    cy.request({ url, auth }).as('req');
    cy.get('@req').then((res) => {
        cy.log(res.status);
        cy.wait(200);
        cy.log(JSON.stringify(res.body));
        cy.wait(200);
        cy.log(JSON.stringify(res.headers));
        cy.wait(3000);
    });
}

describe('debug', () => {
    it('should log /v2/vulnerability-exceptions', () => {
        debugUrl('/v2/vulnerability-exceptions');
    });

    it('should log /v2/reports/configurations', () => {
        debugUrl('/v2/reports/configurations');
    });

    it('should log /v2/reports/configuration-count', () => {
        debugUrl('/v2/reports/configuration-count');
    });

    it('should log /v2/compliance/scan/configurations', () => {
        debugUrl('/v2/compliance/scan/configurations');
    });

    it('should log /v2/compliance/scan/results', () => {
        debugUrl('/v2/compliance/scan/results');
    });

    it('should log /v2/compliance/scan/stats/profile', () => {
        debugUrl('/v2/compliance/scan/stats/profile');
    });

    it('should log /v2/compliance/scan/stats/cluster', () => {
        debugUrl('/v2/compliance/scan/stats/cluster');
    });
});
