import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import {
    hasExpectedHeaderColumns,
    allChecksForEntities,
    // TODO: uncomment the following two imports once we are testing three types of CVEs for cluster
    //       after feature flag for VM updates defaults to ON
    // allCVECheck,
    // allFixableCheck,
} from '../../helpers/vmWorkflowUtils';
import { visitVulnerabilityManagementEntities } from '../../helpers/vulnmanagement/entities';

describe('Clusters list Page and its single entity detail page, and sub list validations ', () => {
    withAuth();

    it('should display all the columns and links expected in clusters list page', () => {
        const usingVMUpdates = hasFeatureFlag('ROX_FRONTEND_VM_UDPATES');

        const columnsToCheck = usingVMUpdates
            ? [
                  'Cluster',
                  'Image CVEs',
                  'Node CVEs',
                  'Platform CVEs',
                  'K8S Version',
                  'Entities',
                  'Policy Status',
                  'Latest Violation',
                  'Risk Priority',
              ]
            : [
                  'Cluster',
                  'CVEs',
                  'K8S Version',
                  'Entities',
                  'Policy Status',
                  'Latest Violation',
                  'Risk Priority',
              ];

        visitVulnerabilityManagementEntities('clusters');
        hasExpectedHeaderColumns(columnsToCheck);

        cy.get(selectors.tableBodyColumn).each(($el) => {
            const columnValue = $el.text().toLowerCase();
            // TODO: replace this helper function for individual entity columns
            //       with one that checks each count in the combined Entities column
            if (columnValue !== 'no namespaces' && columnValue.includes('namespace')) {
                allChecksForEntities(url.list.clusters, 'namespaces');
            }

            if (columnValue !== 'no deployments' && columnValue.includes('deployment')) {
                allChecksForEntities(url.list.clusters, 'deployments');
            }
            // TODO: uncomment and update for three types of CVEs for cluster
            //       after feature flag for VM updates defaults to ON
            // if (columnValue !== 'no cves' && columnValue.includes('cve')) {
            //     allCVECheck(url.list.clusters);
            // }
            // if (columnValue.includes('fixable')) {
            //     allFixableCheck(url.list.clusters);
            // }
        });
    });
});
