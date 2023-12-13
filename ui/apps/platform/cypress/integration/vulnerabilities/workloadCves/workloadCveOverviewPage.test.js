import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';
import {
    applyLocalSeverityFilters,
    selectEntityTab,
    visitWorkloadCveOverview,
} from './WorkloadCves.helpers';

import { selectors } from './WorkloadCves.selectors';

describe('Workload CVE overview page tests', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_VULN_MGMT_WORKLOAD_CVES')) {
            this.skip();
        }
    });

    it('should satisfy initial page load defaults', () => {
        visitWorkloadCveOverview();

        // TODO Test that the default tab is set to "Observed"

        // Check that the CVE entity toggle is selected and Image/Deployment are disabled
        cy.get(selectors.entityTypeToggleItem('CVE')).should('have.attr', 'aria-pressed', 'true');
        cy.get(selectors.entityTypeToggleItem('Image')).should(
            'not.have.attr',
            'aria-pressed',
            'true'
        );
        cy.get(selectors.entityTypeToggleItem('Deployment')).should(
            'not.have.attr',
            'aria-pressed',
            'true'
        );
    });

    it('should correctly handle applied filters across entity tabs', function () {
        if (!hasFeatureFlag('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS')) {
            this.skip();
        }

        visitWorkloadCveOverview();

        // We want to manually test filter application, so clear the default filters
        cy.get(selectors.clearFiltersButton).click();
        cy.get(selectors.isUpdatingTable).should('not.exist');

        const urlBase = '/api/graphql?opname=';

        const entityOpnameMap = {
            CVE: 'getImageCVEList',
            Image: 'getImageList',
            Deployment: 'getDeploymentList',
        };

        const { CVE, Image, Deployment } = entityOpnameMap;

        // Intercept and mock responses as empty, since we don't care about the response
        cy.intercept({ method: 'POST', url: urlBase + CVE }, { data: {} }).as(CVE);
        cy.intercept({ method: 'POST', url: urlBase + Image }, { data: {} }).as(Image);
        cy.intercept({ method: 'POST', url: urlBase + Deployment }, { data: {} }).as(Deployment);

        applyLocalSeverityFilters('Critical');

        // Test that the correct filters are applied for each entity tab, and that the correct
        // search filter is sent in the request for each tab
        Object.entries(entityOpnameMap).forEach(([entity, opname]) => {
            // @ts-ignore
            selectEntityTab(entity);

            // Ensure that only the correct filter chip is present
            cy.get(selectors.filterChipGroupItem('Severity', 'Critical'));
            cy.get(selectors.filterChipGroupItems).should('have.lengthOf', 1);

            // Ensure the correct search filter is present in the request
            cy.wait(`@${opname}`).should((xhr) => {
                expect(xhr.request.body.variables.query).to.contain(
                    'SEVERITY:CRITICAL_VULNERABILITY_SEVERITY'
                );
            });
        });
    });
});
