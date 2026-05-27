import { all, call, fork, put, take } from 'redux-saga/effects';

import { integrationsPath } from 'routePaths';
import { fetchMachineAccessConfigs as serviceFetchMachineAccessConfigs } from 'services/MachineAccessService';
import { actions, types } from 'reducers/machineAccessConfigs';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* getMachineAccessConfigs() {
    try {
        const result = yield call(serviceFetchMachineAccessConfigs);
        yield put(actions.fetchMachineAccessConfigs.success(result.response));
    } catch (error) {
        yield put(actions.fetchMachineAccessConfigs.failure(error));
    }
}

function* watchLocation() {
    yield takeEveryNewlyMatchedLocation(integrationsPath, getMachineAccessConfigs);
}

function* watchFetchRequest() {
    while (true) {
        yield take([types.FETCH_MACHINE_ACCESS_CONFIGS.REQUEST]);
        yield fork(getMachineAccessConfigs);
    }
}

export default function* integrations() {
    yield all([fork(watchLocation), fork(watchFetchRequest)]);
}
