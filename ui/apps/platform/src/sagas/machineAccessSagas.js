import { all, take, call, fork, put, takeLatest } from 'redux-saga/effects';

import { integrationsPath } from 'routePaths';
import * as service from 'services/MachineAccessService';
import { actions, types } from 'reducers/machineAccessConfigs';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* getMachineAccessConfigs() {
    try {
        const result = yield call(service.fetchMachineAccessConfigs);
        yield put(actions.fetchMachineAccessConfigs.success(result.response));
    } catch (error) {
        yield put(actions.fetchMachineAccessConfigs.failure(error));
    }
}

function* deleteMachineAccessConfigs({ ids }) {
    try {
        yield call(service.deleteMachineAccessConfigs, ids);
        yield put(actions.fetchMachineAccessConfigs.request());
    } catch (error) {
        yield put(actions.deleteMachineAccessConfigs.failure(error));
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

function* watchDeleteRequest() {
    yield takeLatest(types.DELETE_MACHINE_ACCESS_CONFIGS, deleteMachineAccessConfigs);
}

export default function* integrations() {
    yield all([fork(watchLocation), fork(watchFetchRequest), fork(watchDeleteRequest)]);
}
