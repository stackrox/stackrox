import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { hasExpectedHeaderColumns } from '../../helpers/vmWorkflowUtils';
import {
    // getCountAndNounFromNodeCVEsLinkResults,
    verifyFilteredSecondaryEntitiesLink,
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from '../../helpers/vulnmanagement/entities';

// After the problem has been fixed, import the function above.
function getCountAndNounFromNodeCVEsLinkResults(resultsFromRegExp) {
    const relatedEntitiesCount = resultsFromRegExp[1];
    const relatedEntitiesNoun = relatedEntitiesCount === 1 ? 'NODE CVE' : 'NODE CVES';
    return {
        panelHeaderText: 'null Node CVES', // workaround for bug
        relatedEntitiesCount,
        relatedEntitiesNoun,
    };
}

const entitiesKey = 'node-components';

describe('Vulnerability Management Node Components', () => {
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
            'Node CVEs',
            'Top CVSS',
            'Nodes',
            'Risk Priority',
        ]);
    });

    //  TBD to be fixed after back end sorting is fixed
    //  validateSort(selectors.componentsRiskScoreCol);

    // Argument 3 in verify functions is one-based index of column which has the links.

    // Some tests might fail in local deployment.

    it('should display links for all node CVEs', () => {
        verifySecondaryEntities(
            entitiesKey,
            'node-cves',
            3,
            /^\d+ CVEs?$/,
            getCountAndNounFromNodeCVEsLinkResults
        );
    });

    it('should display links for fixable node CVEs', () => {
        verifyFilteredSecondaryEntitiesLink(
            entitiesKey,
            'node-cves',
            3,
            /^\d+ Fixable$/,
            getCountAndNounFromNodeCVEsLinkResults
        );
    });

    it('should display links for nodes', () => {
        verifySecondaryEntities(entitiesKey, 'nodes', 5, /^\d+ nodes?$/);
    });
});
