import withAuth from '../../helpers/basicAuth';
import { interceptAndWatchRequests } from '../../helpers/request';
import { selectFilteredWorkflowView, visitViolations } from './Violations.helpers';

describe('Violations - Filtered Workflow Views', () => {
    withAuth();

    const getAlertsRouteMatcher = {
        getAlerts: {
            method: 'GET',
            url: '/v1/alerts?query=*',
        },
    };

    it('should filter the violations table when the "Applications view" is selected', () => {
        interceptAndWatchRequests(getAlertsRouteMatcher).then(({ waitForRequests }) => {
            visitViolations();

            waitForRequests().then((interception) => {
                const queryString = interception.request.query.query as string;

                expect(queryString).to.contain('Entity Type:DEPLOYMENT');
                expect(queryString).to.contain('Platform Component:false');
            });
        });
    });

    it('should filter the violations table when the "Platform view" is selected', () => {
        interceptAndWatchRequests(getAlertsRouteMatcher).then(({ waitForRequests }) => {
            visitViolations();

            waitForRequests();

            selectFilteredWorkflowView('Platform');

            waitForRequests().then((interception) => {
                const queryString = interception.request.query.query as string;

                expect(queryString).to.contain('Entity Type:DEPLOYMENT');
                expect(queryString).to.contain('Platform Component:true');
            });
        });
    });

    it('should filter the violations table when the "Full view" is selected', () => {
        interceptAndWatchRequests(getAlertsRouteMatcher).then(({ waitForRequests }) => {
            visitViolations();

            waitForRequests();

            selectFilteredWorkflowView('All Violations');

            waitForRequests().then((interception) => {
                const queryString = interception.request.query.query as string;

                expect(queryString).to.not.contain('Entity Type:DEPLOYMENT');
                expect(queryString).to.not.contain('Platform Component:true');
                expect(queryString).to.not.contain('Platform Component:false');
            });
        });
    });
});
