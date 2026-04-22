import withAuth from '../../helpers/basicAuth';
import { interceptAndWatchRequests } from '../../helpers/request';
import { selectFilteredWorkflowView, visitRiskDeployments } from './Risk.helpers';

describe('Risk - Filtered Workflow Views', () => {
    withAuth();

    const getDeploymentsRouteMatcher = {
        getDeployments: {
            method: 'GET',
            url: '/v1/deploymentswithprocessinfo*',
        },
    };

    it('should filter the deployments table when the "Applications view" is selected', () => {
        interceptAndWatchRequests(getDeploymentsRouteMatcher).then(({ waitForRequests }) => {
            visitRiskDeployments();

            // Default load is for the "Applications view"
            waitForRequests().then((interception) => {
                const queryString = interception.request.query.query as string;
                expect(queryString).to.equal('Platform Component:false');
            });
        });
    });

    it('should filter the deployments table when the "Platform view" is selected', () => {
        interceptAndWatchRequests(getDeploymentsRouteMatcher).then(({ waitForRequests }) => {
            visitRiskDeployments();

            // Initial load
            waitForRequests();

            selectFilteredWorkflowView('Platform');

            waitForRequests().then((interception) => {
                const queryString = interception.request.query.query as string;
                expect(queryString).to.equal('Platform Component:true');
            });
        });
    });

    it('should filter the deployments table when the "Full view" is selected', () => {
        interceptAndWatchRequests(getDeploymentsRouteMatcher).then(({ waitForRequests }) => {
            visitRiskDeployments();

            // Initial load
            waitForRequests();

            selectFilteredWorkflowView('All Deployments');

            waitForRequests().then((interception) => {
                const queryString = interception.request.query.query as string;
                expect(queryString).to.be.undefined;
            });
        });
    });
});
