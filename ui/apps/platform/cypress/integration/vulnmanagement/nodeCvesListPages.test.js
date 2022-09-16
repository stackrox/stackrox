import { selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { hasExpectedHeaderColumns } from '../../helpers/vmWorkflowUtils';
import {
    verifySecondaryEntities,
    visitVulnerabilityManagementEntities,
} from '../../helpers/vulnmanagement/entities';

const entitiesKey = 'node-cves';

describe('Vulnerability Management Node CVEs', () => {
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
                'Operating System',
                'Fixable',
                'Severity',
                'CVSS Score',
                'Env. Impact',
                'Impact Score',
                'Entities',
                'Discovered Time',
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

    it('should display links for nodes', () => {
        verifySecondaryEntities(entitiesKey, 'nodes', 9, /^\d+ nodes?$/);
    });

    it('should display links for node-components', () => {
        verifySecondaryEntities(entitiesKey, 'node-components', 9, /^\d+ node components?$/);
    });
});
