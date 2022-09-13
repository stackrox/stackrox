import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { hasExpectedHeaderColumns } from '../../helpers/vmWorkflowUtils';
import {
    getCountAndNounFromImageCVEsLinkResults,
    getCountAndNounFromNodeCVEsLinkResults,
    verifyFilteredSecondaryEntitiesLink,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from '../../helpers/vulnmanagement/entities';

function getCountAndNounFromClusterCVEsLinkResults([, count]) {
    return {
        panelHeaderText: `${count} Platform ${count === 1 ? 'CVE' : 'CVES'}`,
        relatedEntitiesCount: count,
        relatedEntitiesNoun: count === 1 ? 'CLUSTER CVE' : 'CLUSTER CVES',
    };
}

const entitiesKey = 'clusters';

describe('Vulnerability Management Clusters', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
            this.skip();
        }
    });

    it('should display all the columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasExpectedHeaderColumns([
            'Cluster',
            'Image CVEs',
            'Node CVEs',
            'Platform CVEs',
            'K8S Version',
            'Entities',
            'Policy Status',
            'Latest Violation',
            'Risk Priority',
        ]);
    });

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

    it('should display links for fixable image CVEs', () => {
        verifyFilteredSecondaryEntitiesLink(
            entitiesKey,
            'image-cves',
            2,
            /^\d+ Fixable$/,
            getCountAndNounFromImageCVEsLinkResults
        );
    });

    it('should display links for all node CVEs', () => {
        verifySecondaryEntities(
            entitiesKey,
            'node-cves',
            3,
            /^\d+ CVEs?$/,
            getCountAndNounFromNodeCVEsLinkResults
        );
    });

    it('should display links for all cluster CVEs', () => {
        verifySecondaryEntities(
            entitiesKey,
            'cluster-cves',
            4,
            /^\d+ CVEs?$/,
            getCountAndNounFromClusterCVEsLinkResults
        );
    });

    it('should display links for namespaces', () => {
        verifySecondaryEntities(entitiesKey, 'namespaces', 6, /^\d+ namespaces?$/);
    });

    it('should display links for deployments', () => {
        verifySecondaryEntities(entitiesKey, 'deployments', 6, /^\d+ deployments?$/);
    });

    it('should display links for nodes', () => {
        verifySecondaryEntities(entitiesKey, 'nodes', 6, /^\d+ nodes?$/);
    });
});
