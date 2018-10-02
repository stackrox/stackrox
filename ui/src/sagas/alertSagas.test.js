import { select, call } from 'redux-saga/effects';
import { expectSaga } from 'redux-saga-test-plan';
import { dynamic } from 'redux-saga-test-plan/providers';

import { selectors } from 'reducers';
import { actions, types } from 'reducers/alerts';
import { types as locationActionTypes } from 'reducers/routes';
import { fetchAlerts, fetchAlert } from 'services/AlertsService';
import { whitelistDeployment } from 'services/PoliciesService';
import saga from './alertSagas';

const alertTypeSearchOptions = [
    {
        value: 'Severity:',
        label: 'Severity:',
        type: 'categoryOption'
    },
    {
        value: 'Low',
        label: 'Low'
    }
];

const alertsSearchQuery = {
    query: 'Severity:Low'
};

const createLocationChange = pathname => ({
    type: locationActionTypes.LOCATION_CHANGE,
    payload: { pathname }
});

describe('Alert Sagas', () => {
    it('should get an unfiltered list of violations on the violations page', () => {
        const violations = ['violation1', 'violation2'];
        const fetchMock = jest.fn().mockReturnValue({ response: violations });

        return expectSaga(saga)
            .provide([
                [select(selectors.getAlertsSearchOptions), []],
                [call(fetchAlerts, { query: '' }), dynamic(fetchMock)]
            ])
            .put(actions.pollAlerts.start())
            .put(actions.fetchAlerts.success(violations))
            .dispatch({ type: types.SET_SEARCH_OPTIONS, payload: { options: [] } })
            .dispatch(createLocationChange('/main/violations'))
            .silentRun();
    });

    it('should get a filtered list of violations on the violations page', () => {
        const violations = ['violation1', 'violation2'];
        const fetchMock = jest.fn().mockReturnValueOnce({ response: violations });

        return expectSaga(saga)
            .provide([
                [select(selectors.getAlertsSearchOptions), alertTypeSearchOptions],
                [call(fetchAlerts, alertsSearchQuery), dynamic(fetchMock)]
            ])
            .put(actions.pollAlerts.start())
            .put(actions.fetchAlerts.success(violations))
            .dispatch({
                type: types.SET_SEARCH_OPTIONS,
                payload: { options: alertTypeSearchOptions }
            })
            .dispatch(createLocationChange('/main/violations'))
            .silentRun();
    });

    it('should re-fetch violations with new alerts search options', () => {
        const violations = ['violation1', 'violation2'];
        const fetchMock = jest.fn().mockReturnValueOnce({ response: violations });

        return expectSaga(saga)
            .provide([
                [select(selectors.getAlertsSearchOptions), alertTypeSearchOptions],
                [call(fetchAlerts, alertsSearchQuery), dynamic(fetchMock)]
            ])
            .put(actions.fetchAlerts.success(violations))
            .dispatch({
                type: types.SET_SEARCH_OPTIONS,
                payload: { options: alertTypeSearchOptions }
            })
            .dispatch(actions.setAlertsSearchOptions(alertTypeSearchOptions))
            .silentRun();
    });

    it('should fetch violations details on the violations page', () => {
        const violation = { id: 'violation1' };
        const fetchMock = jest.fn().mockReturnValueOnce({ response: violation });

        return expectSaga(saga)
            .provide([
                [select(selectors.getAlertsSearchOptions), []],
                [call(fetchAlert, violation.id), dynamic(fetchMock)]
            ])
            .put(actions.fetchAlert.success(violation, { id: violation.id }))
            .dispatch(createLocationChange(`/main/violations/${violation.id}`))
            .silentRun();
    });

    it('should whitelist deployments', () => {
        const response = {
            whitelists: [{ name: 'deploymentName' }]
        };
        const alert = {
            id: '1234',
            policy: {
                id: 'policyId'
            },
            deployment: {
                name: 'deploymentName'
            }
        };
        const fetchWhitelistMock = jest.fn().mockReturnValueOnce({ response });

        return expectSaga(saga)
            .provide([
                [select(selectors.getAlertsSearchOptions), []],
                [select(selectors.getAlert, alert.id), alert],
                [
                    call(whitelistDeployment, alert.policy.id, alert.deployment.name),
                    dynamic(fetchWhitelistMock)
                ]
            ])
            .dispatch(actions.whitelistDeployment.request(alert.id))
            .dispatch({
                type: types.WHITELIST_DEPLOYMENT,
                payload: { options: alertTypeSearchOptions }
            })
            .put(actions.pollAlerts.stop())
            .put(actions.whitelistDeployment.success(response))
            .silentRun();
    });
});
