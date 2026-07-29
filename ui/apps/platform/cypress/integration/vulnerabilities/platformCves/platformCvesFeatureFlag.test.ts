import type { RouteHandler, RouteMatcherOptions } from 'cypress/types/net-stubbing';

import withAuth from '../../../helpers/basicAuth';
import { interceptAndOverrideFeatureFlags } from '../../../helpers/request';
import { visit } from '../../../helpers/visit';
import pf6 from '../../../selectors/pf6';

export function visitPlatformCvesOverviewPage(
    routeMatcherMap?: Record<string, RouteMatcherOptions>,
    staticResponseMap?: Record<string, RouteHandler>
) {
    visit('/main/vulnerabilities/platform-cves', routeMatcherMap, staticResponseMap);
}

describe('Platform CVEs - Feature Flag Gating', () => {
    withAuth();

    describe('when ROX_LEGACY_SCANNER is enabled', () => {
        beforeEach(() => {
            interceptAndOverrideFeatureFlags({ ROX_LEGACY_SCANNER: true });
        });

        it('should show the "Kubernetes components" item in the More Views dropdown', () => {
            visit('/main/vulnerabilities/user-workloads');
            cy.get(`${pf6.menuToggle}:contains("More Views")`).click();
            cy.get(`${pf6.dropdownItem}:contains("Kubernetes components")`).should('exist');
        });

        it('should render the Kubernetes components page when navigated to directly', () => {
            visitPlatformCvesOverviewPage();

            cy.get('h1').should('contain', 'Kubernetes components');
        });
    });

    describe('when ROX_LEGACY_SCANNER is disabled', () => {
        beforeEach(() => {
            interceptAndOverrideFeatureFlags({ ROX_LEGACY_SCANNER: false });
        });

        it('should not show the "Kubernetes components" item in the More Views dropdown', () => {
            visit('/main/vulnerabilities/user-workloads');
            cy.get(`${pf6.menuToggle}:contains("More Views")`).click();
            cy.get(`${pf6.dropdownItem}:contains("Kubernetes components")`).should('not.exist');
        });

        it('should show a disabled feature page when navigated to directly', () => {
            visitPlatformCvesOverviewPage();

            cy.get('h1').should('contain', 'Kubernetes components');
            cy.get('body').should('contain', 'disabled by your administrator');
            cy.get('a:contains("Go to Vulnerability Management")').should('exist');
        });
    });
});
