import { visitFromConsoleLeftNavExpandable } from '../../helpers/nav';
import { withOcpAuth } from '../../helpers/ocpAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { assertVisibleTableColumns } from '../../helpers/tableHelpers';
import { selectors } from '../../integration/vulnerabilities/vulnerabilities.selectors';
import { selectProject } from '../../helpers/ocpConsole';
import { assertSearchEntities } from '../../integration/vulnerabilities/workloadCves/WorkloadCves.helpers';

describe('Security vulnerabilities page', () => {
    it('should display only the expected table columns for each entity type', () => {
        withOcpAuth();
        visitFromConsoleLeftNavExpandable('Security', 'Vulnerabilities');

        // Check CVE table columns
        const expectedCveTableColumns = [
            'Row expansion',
            'CVE',
            'Images by severity',
            'Top CVSS',
            hasFeatureFlag('ROX_SCANNER_V4') ? 'Top NVD CVSS' : null,
            hasFeatureFlag('ROX_SCANNER_V4') ? 'EPSS probability' : null,
            'First discovered',
            'Published',
        ].filter((column) => column !== null);
        assertVisibleTableColumns('table', expectedCveTableColumns);

        // Check Image table columns
        const expectedImageTableColumns = [
            'Image',
            'CVEs by severity',
            'Operating system',
            'Deployments',
            'Age',
            'Scan time',
        ];
        cy.get(selectors.entityTypeToggleItem('Image')).click();
        assertVisibleTableColumns('table', expectedImageTableColumns);

        // Check Deployment table columns
        const expectedDeploymentTableColumns = [
            'Deployment',
            'CVEs by severity',
            'Namespace',
            'Images',
            'First discovered',
        ];
        cy.get(selectors.entityTypeToggleItem('Deployment')).click();
        assertVisibleTableColumns('table', expectedDeploymentTableColumns);
    });

    it('should restrict the UI based on the status of the selected project', () => {
        withOcpAuth();
        visitFromConsoleLeftNavExpandable('Security', 'Vulnerabilities');

        // Namespace is available by default when viewing "All projects"
        assertSearchEntities(['CVE', 'Image', 'Image component', 'Deployment', 'Namespace']);

        selectProject('stackrox');

        // Namespace is not available when viewing a specific project
        assertSearchEntities(['CVE', 'Image', 'Image component', 'Deployment']);

        // Check Deployment table columns, expect `Namespace` to be missing
        const expectedDeploymentTableColumns = [
            'Deployment',
            'CVEs by severity',
            'Images',
            'First discovered',
        ];
        cy.get(selectors.entityTypeToggleItem('Deployment')).click();
        assertVisibleTableColumns('table', expectedDeploymentTableColumns);
    });
});
