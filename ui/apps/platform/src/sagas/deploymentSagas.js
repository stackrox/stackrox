import { all, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import { dashboardPath, policiesPath } from 'routePaths';
import { fetchDeploymentsLegacy as fetchDeployments } from 'services/DeploymentsService';
import { actions } from 'reducers/deployments';
import { types as dashboardTypes } from 'reducers/dashboard';
import { selectors } from 'reducers';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

export function* getDeployments({ options = [] }) {
    try {
        const result = yield call(fetchDeployments, options);
        yield put(actions.fetchDeployments.success(result.response, { options }));
    } catch (error) {
        yield put(actions.fetchDeployments.failure(error));
    }
}

function* filterDashboardPageBySearch() {
    const options = yield select(selectors.getDashboardSearchOptions);
    if (options.length && options[options.length - 1].type) {
        return;
    }
    yield fork(getDeployments, { options });
}

function* filterPoliciesPageBySearch() {
    const options = yield select(selectors.getPoliciesSearchOptions);
    if (options.length && options[options.length - 1].type) {
        return;
    }
    yield fork(getDeployments, { options });
}

function* watchDashboardSearchOptions() {
    yield takeLatest(dashboardTypes.SET_SEARCH_OPTIONS, filterDashboardPageBySearch);
}

export default function* deployments() {
    yield all([
        takeEveryNewlyMatchedLocation(dashboardPath, filterDashboardPageBySearch),
        takeEveryNewlyMatchedLocation(policiesPath, filterPoliciesPageBySearch),
        fork(watchDashboardSearchOptions),
    ]);
}
