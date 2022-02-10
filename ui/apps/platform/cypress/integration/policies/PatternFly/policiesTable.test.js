import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { visitPolicies } from '../../../helpers/policiesPatternFly';

describe('Policies table', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_POLICIES_PATTERNFLY')) {
            this.skip();
        }
    });

    it('should have id', () => {
        visitPolicies();

        cy.get('[id="policies-table"]');
    });
});
