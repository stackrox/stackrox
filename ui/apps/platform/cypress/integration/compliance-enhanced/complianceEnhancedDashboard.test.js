import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    visitComplianceEnhancedDashboard,
    visitComplianceEnhancedFromLeftNav,
    statusDashboardPath,
} from './ComplianceEnhanced.helpers';

describe('Compliance Dashboard', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_COMPLIANCE_ENHANCEMENTS')) {
            this.skip();
        }
    });

    it('should visit using the left nav', () => {
        visitComplianceEnhancedFromLeftNav();

        cy.location('pathname').should('eq', statusDashboardPath);

        cy.title().should('match', getRegExpForTitleWithBranding('Compliance Status Dashboard'));
    });

    it('should have expected elements on the status page', () => {
        visitComplianceEnhancedDashboard();

        cy.location('pathname').should('eq', statusDashboardPath);

        cy.title().should('match', getRegExpForTitleWithBranding('Compliance Status Dashboard'));

        cy.get('.pf-c-card__header:contains("Compliance by cluster")');
        cy.get('.pf-c-card__header:contains("Compliance by profile")');

        cy.get('th[scope="col"]:contains("Scan")');
        cy.get('th[scope="col"]:contains("Clusters")');
        cy.get('th[scope="col"]:contains("Profiles")');
        cy.get('th[scope="col"]:contains("Failing Controls")');
        cy.get('th[scope="col"]:contains("Last Scanned")');
    });
});
