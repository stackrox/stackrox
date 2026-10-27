import CvssFormatted from './CvssFormatted';

describe(Cypress.spec.relative, () => {
    [null, undefined, NaN, Infinity, -Infinity].forEach((cvss) => {
        it(`should display Not available for ${String(cvss)}`, () => {
            cy.mount(<CvssFormatted cvss={cvss} scoreVersion="V3" />);

            cy.findByText('Not available').should('exist');
        });
    });

    it('should format an available score and its version', () => {
        cy.mount(<CvssFormatted cvss={9.4300001} scoreVersion="V3" />);

        cy.findByText('9.4 (V3)').should('exist');
    });

    it('should format zero as an available score', () => {
        cy.mount(<CvssFormatted cvss={0} />);

        cy.findByText('0.0').should('exist');
    });

    it('should preserve the unavailable marker for an unknown score version', () => {
        cy.mount(<CvssFormatted cvss={9.4} scoreVersion="UNKNOWN_VERSION" />);

        cy.findByText('Not available').should('exist');
    });
});
