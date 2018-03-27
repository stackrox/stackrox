import { take, call, fork, put } from 'redux-saga/effects';

import fetchDeployments from 'services/RiskService';
import { actions } from 'reducers/risk';
import { types as locationActionTypes } from 'reducers/routes';

const riskPath = '/main/risk';

export function* getDeployments() {
    try {
        const result = yield call(fetchDeployments);
        yield put(actions.fetchDeployments.success(result.response));
    } catch (error) {
        yield put(actions.fetchDeployments.failure(error));
    }
}

export function* watchLocation() {
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (location && location.pathname && location.pathname.startsWith(riskPath)) {
            yield fork(getDeployments);
        }
    }
}

export default function* deployments() {
    yield fork(watchLocation);
}
