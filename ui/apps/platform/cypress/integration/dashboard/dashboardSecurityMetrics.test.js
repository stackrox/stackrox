import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { visitMainDashboardPF } from '../../helpers/main';

describe('Dashboard security metrics phase one action widgets', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_SECURITY_METRICS_PHASE_ONE')) {
            this.skip();
        }
    });

    it('should visit patternfly dashboard', () => {
        visitMainDashboardPF();

        cy.get("h1:contains('Dashboard')");
    });
});
