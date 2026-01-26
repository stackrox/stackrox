import { visitFromConsoleLeftNavExpandable } from '../../helpers/nav';
import { withOcpAuth } from '../../helpers/ocpAuth';
import { interceptAndWatchRequests } from '../../helpers/request';
import pf6 from '../../selectors/pf6';
import {
    acsAuthNamespaceHeader,
    getImageCVEListRoute,
    getImageCVEListRouteMatcher,
    routeMatcherMapForBasePlugin,
} from '../routes';

function visitProjectSecurityTabAndCheckAuthHeaders(
    visitFunction: () => void,
    resourceIconTitle: string
) {
    interceptAndWatchRequests({
        ...routeMatcherMapForBasePlugin,
        [getImageCVEListRoute]: getImageCVEListRouteMatcher,
    }).then(({ waitForRequests }) => {
        withOcpAuth();
        visitFunction();

        cy.get(pf6.menuToggle).contains('Requester').click();
        cy.get(pf6.menuItem).contains('Name').click();
        cy.get('input[aria-label="Name filter"]').type('stackrox');
        cy.get(`[title="${resourceIconTitle}"] + a`).contains('stackrox').click();
        cy.get(pf6.tabButton).contains('Security').click();

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

describe('Project Security Tabs', () => {
    describe('Project Security Tab via Home -> Projects', () => {
        it('should send the correct auth headers for namespace scoped requests on project security tab', () => {
            visitProjectSecurityTabAndCheckAuthHeaders(() => {
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
    });
});
