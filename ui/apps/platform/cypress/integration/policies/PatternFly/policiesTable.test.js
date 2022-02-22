import withAuth from '../../../helpers/basicAuth';
import { visitPolicies } from '../../../helpers/policiesPatternFly';

describe('Policies table', () => {
    withAuth();

    it('should have id', () => {
        visitPolicies();

        cy.get('[id="policies-table"]');
    });
});
