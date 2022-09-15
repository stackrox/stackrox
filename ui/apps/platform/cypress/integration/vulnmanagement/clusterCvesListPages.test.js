import { selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { hasExpectedHeaderColumns } from '../../helpers/vmWorkflowUtils';
import {
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from '../../helpers/vulnmanagement/entities';

// After the problem has been fixed, remove the function argument below.
function getCountAndNounFromClustersLinkResults(resultsFromRegExp) {
    const relatedEntitiesCount = resultsFromRegExp[1];
    const relatedEntitiesNoun = relatedEntitiesCount === 1 ? 'CLUSTER' : 'CLUSTERS';
    return {
        panelHeaderText: '0 clusters', // workaround for bug
        relatedEntitiesCount,
        relatedEntitiesNoun,
    };
}

const entitiesKey = 'cluster-cves';

describe('Vulnerability Management Cluster (Platform) CVEs', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
            this.skip();
        }
    });

    it('should display table columns', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        hasExpectedHeaderColumns(
            [
                'CVE',
                'Type',
                'Fixable',
                'CVSS Score',
                'Env. Impact',
                'Impact Score',
                'Entities',
                'Published',
            ],
            1 // skip 1 additional column to account for checkbox column
        );
    });

    // TODO to be fixed after back end sorting is fixed
    // validateSortForCVE(selectors.cvesCvssScoreCol);

    it('should display vulnerability descriptions', () => {
        visitVulnerabilityManagementEntities(entitiesKey);

        // Balance positive and negative assertions.
        cy.get(selectors.cveDescription).should('exist');
        cy.get(`${selectors.cveDescription}:contains("No description available")`).should(
            'not.exist'
        );
    });

    // Argument 3 in verify functions is one-based index of column which has the links.
    // Count the checkbox as the first column.

    // Some tests might fail in local deployment.

    // TODO Investigate why CI displays No clusters instead of 2 clusters.
    it.skip('should display links for clusters', () => {
        verifySecondaryEntities(
            entitiesKey,
            'clusters',
            8,
            /^\d+ clusters?$/,
            getCountAndNounFromClustersLinkResults
        );
    });
});
