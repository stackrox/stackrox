import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { hasExpectedHeaderColumns } from '../../helpers/vmWorkflowUtils';
import {
    getCountAndNounFromImageCVEsLinkResults,
    verifyFilteredSecondaryEntitiesLink,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from '../../helpers/vulnmanagement/entities';

const entitiesKey = 'image-components';

describe('Vulnerability Management Image Components', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
            this.skip();
        }
    });

    it('should display table columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasExpectedHeaderColumns([
            'Component',
            'Operating System',
            'CVEs',
            'Fixed In',
            'Top CVSS',
            'Images',
            'Deployments',
            'Risk Priority',
        ]);
    });

    //  TBD to be fixed after back end sorting is fixed
    //  validateSort(selectors.componentsRiskScoreCol);

    // Argument 3 in verify functions is one-based index of column which has the links.

    // Some tests might fail in local deployment.

    it('should display links for all image CVEs', () => {
        verifySecondaryEntities(
            entitiesKey,
            'image-cves',
            3,
            /^\d+ CVEs?$/,
            getCountAndNounFromImageCVEsLinkResults
        );
    });

    it('should display links for fixable image CVEs', () => {
        verifyFilteredSecondaryEntitiesLink(
            entitiesKey,
            'image-cves',
            3,
            /^\d+ Fixable$/,
            getCountAndNounFromImageCVEsLinkResults
        );
    });

    it('should display links for images', () => {
        verifySecondaryEntities(entitiesKey, 'images', 6, /^\d+ images?$/);
    });

    it('should display links for deployments', () => {
        verifySecondaryEntities(entitiesKey, 'deployments', 7, /^\d+ deployments?$/);
    });
});
