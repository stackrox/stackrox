import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import {
    visitSystemHealth,
    visitSystemHealthFromLeftNav,
} from '../../helpers/systemHealthPatternFly';
import navSelectors from '../../selectors/navigation';

describe('System Health general', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_SYSTEM_HEALTH_PF')) {
            this.skip();
        }
    });

    it.skip('should visit from left nav', () => {
        visitSystemHealthFromLeftNav();
    });

    it('should have selected item in left nav', () => {
        visitSystemHealth();

        cy.get(`${navSelectors.navExpandable}:contains("Platform Configuration")`);
        cy.get(`${navSelectors.nestedNavLinks}:contains("System Health")`).should(
            'have.class',
            'pf-m-current'
        );
    });
});
