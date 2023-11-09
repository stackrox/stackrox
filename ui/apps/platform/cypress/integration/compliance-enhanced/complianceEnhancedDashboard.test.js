import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    visitComplianceEnhancedDashboard,
    visitComplianceEnhancedFromLeftNav,
    visitComplianceEnhancedScanConfigsFromLeftNav,
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

        cy.title().should('match', getRegExpForTitleWithBranding('Compliance Status Dashboard'));
    });

    it('should have expected elements on the status page', () => {
        visitComplianceEnhancedDashboard();

        cy.title().should('match', getRegExpForTitleWithBranding('Compliance Status Dashboard'));

        cy.get('.pf-c-card__header:contains("Compliance by cluster")');
        cy.get('.pf-c-card__header:contains("Compliance by profile")');

        cy.get('th[scope="col"]:contains("Scan")');
        cy.get('th[scope="col"]:contains("Clusters")');
        cy.get('th[scope="col"]:contains("Profiles")');
        cy.get('th[scope="col"]:contains("Failing Controls")');
        cy.get('th[scope="col"]:contains("Last Scanned")');
    });

    it('should visit scan configurations scheduling from the left nav', () => {
        visitComplianceEnhancedScanConfigsFromLeftNav();

        cy.title().should('match', getRegExpForTitleWithBranding('Scan Schedules'));
    });

    it('should have expected elements on the scans page', () => {
        visitComplianceEnhancedScanConfigsFromLeftNav();

        cy.title().should('match', getRegExpForTitleWithBranding('Scan Schedules'));

        cy.get('th[scope="col"]:contains("Name")');
        cy.get('th[scope="col"]:contains("Schedule")');
        cy.get('th[scope="col"]:contains("Last run")');
        cy.get('th[scope="col"]:contains("Clusters")');
        cy.get('th[scope="col"]:contains("Profiles")');

        cy.get('h2:contains("No scan schedules")');

        cy.get('.pf-c-toolbar__content a:contains("Create scan schedule")').click();
        cy.location('search').should('eq', '?action=create');

        cy.go('back');

        cy.get('.pf-c-empty-state__content a:contains("Create scan schedule")').click();
        cy.location('search').should('eq', '?action=create');
    });
});
