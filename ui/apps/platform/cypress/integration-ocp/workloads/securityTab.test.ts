import { visitFromConsoleLeftNavExpandable } from '../../helpers/nav';
import { withOcpAuth } from '../../helpers/ocpAuth';
import { selectProject } from '../../helpers/ocpConsole';
import { interceptAndWatchRequests } from '../../helpers/request';
import {
    acsAuthNamespaceHeader,
    deploymentListRoute,
    deploymentListRouteMatcher,
    getCVEsForDeploymentRoute,
    getCVEsForDeploymentRouteMatcher,
    routeMatcherMapForBasePlugin,
} from '../routes';

describe('Workloads - Security tab', () => {
    it('should send the correct auth headers for namespace scoped requests on workload security tab', () => {
        interceptAndWatchRequests({
            ...routeMatcherMapForBasePlugin,
            [deploymentListRoute]: deploymentListRouteMatcher,
            [getCVEsForDeploymentRoute]: getCVEsForDeploymentRouteMatcher,
        }).then(({ waitForRequests }) => {
            withOcpAuth();
            visitFromConsoleLeftNavExpandable('Workloads', 'Deployments');
            selectProject('stackrox');

            cy.get('input[aria-label="Name filter"]').type('central-db');
            cy.get('[title="Deployment"] + a').contains('central-db').click();
            cy.get('[role="tab"]').contains('Security').click();

            waitForRequests([]).then(
                ([
                    metadataRequest,
                    featureFlagsRequest,
                    publicConfigRequest,
                    deploymentListRequest,
                    getCVEsForDeploymentRequest,
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
                    expect(deploymentListRequest.request.headers[acsAuthNamespaceHeader]).to.equal(
                        'stackrox'
                    );
                    expect(
                        getCVEsForDeploymentRequest.request.headers[acsAuthNamespaceHeader]
                    ).to.equal('stackrox');
                }
            );
        });
    });
});
