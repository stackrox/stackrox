import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import { visitPolicies } from '../../../helpers/policiesPatternFly';

describe('Policy table', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_POLICIES_PATTERNFLY')) {
            this.skip();
        }
    });

    it('should have the temporary placeholder', () => {
        visitPolicies();

        cy.get('div[data-testid="policies-placeholder"]:Contains("Policies")');
    });
});
