import { visitFromConsoleLeftNavExpandable } from '../../helpers/nav';
import { withOcpAuth } from '../../helpers/ocpAuth';
import { assertVisibleTableColumns } from '../../helpers/tableHelpers';
import { interceptAndOverrideFeatureFlags, interceptAndWatchRequests } from '../../helpers/request';
import pf6 from '../../selectors/pf6';
import { selectors } from '../../integration/vulnerabilities/vulnerabilities.selectors';
import {
    acsAuthNamespaceHeader,
    featureFlagsRouteMatcher,
    getImageCVEListRoute,
    getImageCVEListRouteMatcher,
    routeMatcherMapForBasePlugin,
} from '../routes';
import { filterByField } from '../../helpers/ocpConsole';

function visitStackroxProjectSecurityTab(visitFunction: () => void, resourceIconTitle: string) {
    withOcpAuth();
    visitFunction();

    filterByField('Name', 'stackrox');
    cy.get(`[title="${resourceIconTitle}"] + a`).contains('stackrox').click();
    cy.get(pf6.tabButton).contains('Security').click();
}

function visitProjectSecurityTabAndCheckAuthHeaders(
    visitFunction: () => void,
    resourceIconTitle: string
) {
    interceptAndWatchRequests({
        ...routeMatcherMapForBasePlugin,
        [getImageCVEListRoute]: getImageCVEListRouteMatcher,
    }).then(({ waitForRequests }) => {
        visitStackroxProjectSecurityTab(visitFunction, resourceIconTitle);

        waitForRequests([]).then(
            ([
                metadataRequest,
                featureFlagsRequest,
                publicConfigRequest,
                getImageCVEListRequest,
            ]) => {
                expect(metadataRequest.request.headers).not.to.have.property(
                    acsAuthNamespaceHeader
                );
                expect(featureFlagsRequest.request.headers).not.to.have.property(
                    acsAuthNamespaceHeader
                );
                expect(publicConfigRequest.request.headers).not.to.have.property(
                    acsAuthNamespaceHeader
                );
                expect(getImageCVEListRequest.request.headers[acsAuthNamespaceHeader]).to.equal(
                    'stackrox'
                );
            }
        );
    });
}

function visitProjectSecurityTabAndCheckColumns(
    visitFunction: () => void,
    resourceIconTitle: string
) {
    // The CVE column set depends on ROX_SCANNER_V4, so pin the flag in the plugin's own response
    // instead of trusting the test environment to have exported the same flag (ROX-36808).
    interceptAndOverrideFeatureFlags({ ROX_SCANNER_V4: false }, featureFlagsRouteMatcher.url);
    visitStackroxProjectSecurityTab(visitFunction, resourceIconTitle);

    // Check CVE table columns
    const expectedCveTableColumns = [
        'Row expansion',
        'CVE',
        'Images by severity',
        'Top CVSS',
        'First discovered',
        'Published',
    ];
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

    // Check Deployment table columns - Namespace column should be hidden in project-scoped view
    const expectedDeploymentTableColumns = [
        'Deployment',
        'CVEs by severity',
        'Images',
        'First discovered',
    ];
    cy.get(selectors.entityTypeToggleItem('Deployment')).click();
    assertVisibleTableColumns('table', expectedDeploymentTableColumns);
}

describe('Project Security Tabs', () => {
    describe('Project Security Tab via Home -> Projects', () => {
        it('should send the correct auth headers for namespace scoped requests on project security tab', () => {
            visitProjectSecurityTabAndCheckAuthHeaders(() => {
                visitFromConsoleLeftNavExpandable('Home', 'Projects');
            }, 'Project');
        });

        it('should display only the expected table columns for each entity type', () => {
            visitProjectSecurityTabAndCheckColumns(() => {
                visitFromConsoleLeftNavExpandable('Home', 'Projects');
            }, 'Project');
        });
    });

    describe('Project Security Tab via Administration -> Namespaces', () => {
        it('should send the correct auth headers for namespace scoped requests on namespace security tab', () => {
            visitProjectSecurityTabAndCheckAuthHeaders(() => {
                visitFromConsoleLeftNavExpandable('Administration', 'Namespaces');
            }, 'Namespace');
        });

        it('should display only the expected table columns for each entity type', () => {
            visitProjectSecurityTabAndCheckColumns(() => {
                visitFromConsoleLeftNavExpandable('Administration', 'Namespaces');
            }, 'Namespace');
        });
    });
});
