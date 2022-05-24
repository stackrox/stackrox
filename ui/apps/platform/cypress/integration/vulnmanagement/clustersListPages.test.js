import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import {
    hasExpectedHeaderColumns,
    allChecksForEntities,
    allCVECheck,
    allFixableCheck,
} from '../../helpers/vmWorkflowUtils';
import { visitVulnerabilityManagementEntities } from '../../helpers/vulnmanagement/entities';

describe('Clusters list Page and its single entity detail page, and sub list validations ', () => {
    withAuth();

    it('should display all the columns and links expected in clusters list page', () => {
        visitVulnerabilityManagementEntities('clusters');
        hasExpectedHeaderColumns([
            'Cluster',
            'CVEs',
            'K8S Version',
            'Namespaces',
            'Deployments',
            'Policy Status',
            'Latest Violation',
            'Risk Priority',
        ]);
        cy.get(selectors.tableBodyColumn).each(($el) => {
            const columnValue = $el.text().toLowerCase();
            if (columnValue !== 'no namespaces' && columnValue.includes('namespace')) {
                allChecksForEntities(url.list.clusters, 'namespaces');
            }

            if (columnValue !== 'no deployments' && columnValue.includes('deployment')) {
                allChecksForEntities(url.list.clusters, 'deployments');
            }
            if (columnValue !== 'no cves' && columnValue.includes('cve')) {
                allCVECheck(url.list.clusters);
            }
            if (columnValue.includes('fixable')) {
                allFixableCheck(url.list.clusters);
            }
        });
    });
});
