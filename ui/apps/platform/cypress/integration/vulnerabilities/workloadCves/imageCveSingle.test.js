import withAuth from '../../../helpers/basicAuth';
import { verifyColumnManagement } from '../../../helpers/tableHelpers';
import {
    getRouteMatcherMapForGraphQL,
    interactAndWaitForResponses,
} from '../../../helpers/request';
import { selectEntityTab, visitWorkloadCveOverview } from './WorkloadCves.helpers';
import { compoundFiltersSelectors } from '../../../helpers/compoundFilters';

describe('Workload CVE Image CVE Single page', () => {
    withAuth();

    function visitFirstCve() {
        visitWorkloadCveOverview();

        const routeMatcherMap = getRouteMatcherMapForGraphQL([
            'getImageCveMetadata',
            'getImageCveSummaryData',
            'getImagesForCVE',
        ]);
        const staticResponseMap = {
            getImageCveMetadata: {
                fixture: 'vulnerabilities/workloadCves/getImageCveMetadata.json',
            },
            getImageCveSummaryData: {
                fixture: 'vulnerabilities/workloadCves/getImageCveSummaryData.json',
            },
            getImagesForCVE: {
                fixture: 'vulnerabilities/workloadCves/getImagesForCVE.json',
            },
        };

        interactAndWaitForResponses(
            () => {
                cy.get('tbody tr td[data-label="CVE"] a').first().click();
            },
            routeMatcherMap,
            staticResponseMap
        );
    }

    it('should correctly handle ImageCVE single page specific behavior', () => {
        visitFirstCve();

        // Check that only applicable resource menu items are present in the toolbar
        cy.get(compoundFiltersSelectors.entityMenuToggle).click();
        cy.get(compoundFiltersSelectors.entityMenuItem).contains('CVE').should('not.exist');
        cy.get(compoundFiltersSelectors.entityMenuItem).contains('Image');
        cy.get(compoundFiltersSelectors.entityMenuItem).contains('Image component');
        cy.get(compoundFiltersSelectors.entityMenuItem).contains('Deployment');
        cy.get(compoundFiltersSelectors.entityMenuItem).contains('Cluster');
        cy.get(compoundFiltersSelectors.entityMenuItem).contains('Namespace');
        cy.get(compoundFiltersSelectors.entityMenuToggle).click();
    });

    describe('Column management tests', () => {
        it('should allow the user to hide and show columns on the Images tab', () => {
            visitFirstCve();
            verifyColumnManagement({ tableSelector: 'table' });
        });

        it('should allow the user to hide and show columns on the Deployments tab', () => {
            visitFirstCve();
            selectEntityTab('Deployment');
            verifyColumnManagement({ tableSelector: 'table' });
        });
    });
});
