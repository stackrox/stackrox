import { all, take, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import fetchDeployments from 'services/RiskService';
import { actions, types } from 'reducers/risk';
import { types as locationActionTypes } from 'reducers/routes';
import { selectors } from 'reducers';

const riskPath = '/main/risk';
const dashboardPath = '/main/dashboard';

export function* getDeployments() {
    try {
        const searchQuery = yield select(selectors.getDeploymentsSearchQuery);
        const filters = {
            query: searchQuery
        };
        const result = yield call(fetchDeployments, filters);
        yield put(actions.fetchDeployments.success(result.response));
    } catch (error) {
        yield put(actions.fetchDeployments.failure(error));
    }
}

function* watchDeploymentsSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, getDeployments);
}

export function* watchLocation() {
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (
            location &&
            location.pathname &&
            (location.pathname.startsWith(riskPath) || location.pathname.startsWith(dashboardPath))
        ) {
            yield fork(getDeployments);
        }
    }
}

export default function* deployments() {
    yield all([fork(watchLocation), fork(watchDeploymentsSearchOptions)]);
}
