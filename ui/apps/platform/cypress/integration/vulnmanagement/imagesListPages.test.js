import { selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { hasExpectedHeaderColumns } from '../../helpers/vmWorkflowUtils';
import {
    getCountAndNounFromImageCVEsLinkResults,
    verifyFixableCVEsLinkAndRiskAcceptanceTabs,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from '../../helpers/vulnmanagement/entities';

const entitiesKey = 'images';

describe('Vulnerability Management Images', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
            this.skip();
        }
    });

    it('should display table columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasExpectedHeaderColumns([
            'Image',
            'CVEs',
            'Top CVSS',
            'Created',
            'Scan Time',
            'Image OS',
            'Image Status',
            'Entities',
            'Risk Priority',
        ]);
    });

    //  TBD to be fixed after back end sorting is fixed
    //  validateSort(selectors.riskScoreCol);

    // Argument 3 in verify functions is one-based index of column which has the links.

    // Some tests might fail in local deployment.

    it('should display links for all image CVEs', () => {
        verifySecondaryEntities(
            entitiesKey,
            'image-cves',
            2,
            /^\d+ CVEs?$/,
            getCountAndNounFromImageCVEsLinkResults
        );
    });

    it('should display links for fixable image CVEs and also Risk Acceptance tabs', () => {
        verifyFixableCVEsLinkAndRiskAcceptanceTabs(
            entitiesKey,
            'image-cves',
            2,
            /^\d+ Fixable$/,
            getCountAndNounFromImageCVEsLinkResults
        );
    });

    it('should display links for deployments', () => {
        verifySecondaryEntities(entitiesKey, 'deployments', 8, /^\d+ deployments?$/);
    });

    it('should display links for image-components', () => {
        verifySecondaryEntities(entitiesKey, 'image-components', 8, /^\d+ image components?$/);
    });

    it('should show entity icon, not back button, if there is only one item on the side panel stack', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        cy.get(`${selectors.deploymentCountLink}:eq(0)`).click({ force: true });
        cy.wait(1000);
        cy.get(selectors.backButton).should('exist');
        cy.get(selectors.entityIcon).should('not.exist');

        cy.get(selectors.backButton).click();
        cy.get(selectors.backButton).should('not.exist');
        cy.get(selectors.entityIcon).should('exist');
    });
});
