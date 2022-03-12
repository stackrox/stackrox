import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';

describe('Smoke test for vulnmanagement', () => {
    withAuth();

    describe('with old single menu item', () => {
        before(function beforeHook() {
            if (hasFeatureFlag('ROX_VULN_REPORTING')) {
                this.skip();
            }
        });

        it('VulnManagement tile link is present and lands on dashboard page', () => {
            cy.visit('/main/dashboard');
            cy.get(selectors.vulnManagementNavLink).click({ force: true });
            cy.url().should('contain', url.dashboard);
        });
    });

    describe('with old single menu item', () => {
        before(function beforeHook() {
            if (!hasFeatureFlag('ROX_VULN_REPORTING')) {
                this.skip();
            }
        });

        it('VulnManagement tile link is present and lands on dashboard page', () => {
            cy.visit('/main/dashboard');
            cy.get(selectors.vulnManagementExpandableNavLink).click({ force: true });
            cy.get(selectors.vulnManagementExpandedDashboardNavLink).click({ force: true });
            cy.url().should('contain', url.dashboard);
        });
    });
});
