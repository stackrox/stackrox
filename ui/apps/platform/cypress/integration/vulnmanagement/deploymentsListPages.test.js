import { url, selectors } from '../../constants/VulnManagementPage';
import withAuth from '../../helpers/basicAuth';
import {
    hasExpectedHeaderColumns,
    allChecksForEntities,
    allCVECheck,
    // uncomment after the issue fix  - allFixableCheck
} from '../../helpers/vmWorkflowUtils';
import { visitVulnerabilityManagementEntities } from '../../helpers/vulnmanagement/entities';

describe('Deployments list Page and its entity detail page , (related entities) sub list  validations ', () => {
    withAuth();

    it('should display all the columns and links expected in deployments list page', () => {
        visitVulnerabilityManagementEntities('deployments');
        hasExpectedHeaderColumns([
            'Deployment',
            'CVEs',
            'Latest Violation',
            'Policy Status',
            'Cluster',
            'Namespace',
            'Images',
            'Risk Priority',
        ]);
        cy.get(selectors.tableBodyColumn).each(($el) => {
            const columnValue = $el.text().toLowerCase();
            if (columnValue !== 'no failing policies' && columnValue.includes('polic')) {
                allChecksForEntities(url.list.deployments, 'Polic');
            }
            if (columnValue !== 'no images' && columnValue.includes('image')) {
                allChecksForEntities(url.list.deployments, 'image');
            }
            /* TBD - remove comment after issue fixed : if (columnValue !== 'no cves' && columnValue.includes('fixable'))
                allFixableCheck(url.list.deployments); */
            if (columnValue !== 'no cves' && columnValue.includes('cve')) {
                allCVECheck(url.list.deployments);
            }
        });
        //  TBD to be fixed after back end sorting is fixed
        //  validateSort(selectors.riskScoreCol);
    });
});
