import { visitFromConsoleLeftNavExpandable } from '../../helpers/nav';
import { withOcpAuth } from '../../helpers/ocpAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { assertVisibleTableColumns } from '../../helpers/tableHelpers';
import { interceptAndWatchRequests } from '../../helpers/request';
import pf6 from '../../selectors/pf6';
import { selectors } from '../../integration/vulnerabilities/vulnerabilities.selectors';
import {
    acsAuthNamespaceHeader,
    getImageCVEListRoute,
    getImageCVEListRouteMatcher,
    routeMatcherMapForBasePlugin,
} from '../routes';

function visitStackroxProjectSecurityTab(visitFunction: () => void, resourceIconTitle: string) {
    withOcpAuth();
    visitFunction();

    cy.get(pf6.menuToggle).contains('Requester').click();
    cy.get(pf6.menuItem).contains('Name').click();
    cy.get('input[aria-label="Name filter"]').type('stackrox');
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
    visitStackroxProjectSecurityTab(visitFunction, resourceIconTitle);

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
