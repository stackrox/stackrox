import { all, takeLatest, call, fork, put } from 'redux-saga/effects';
import Raven from 'raven-js';

import { systemConfigPath } from 'routePaths';
import { actions, types } from 'reducers/telemetryConfig';
import { actions as notificationActions } from 'reducers/notifications';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import { fetchTelemetryConfig, saveTelemetryConfig } from '../services/TelemetryService';

function* getTelemetryConfig() {
    try {
        const { response } = yield call(fetchTelemetryConfig);
        yield put(actions.fetchTelemetryConfig.success(response));
    } catch (error) {
        yield put(actions.fetchTelemetryConfig.failure(error));
    }
}

function* updateTelemetryConfig(action) {
    try {
        yield call(saveTelemetryConfig, action.telemetryConfig);
        yield fork(getTelemetryConfig);
    } catch (error) {
        if (error.response) {
            yield put(notificationActions.addNotification(error.response.data.error));
            yield put(notificationActions.removeOldestNotification());
        } else {
            // TODO-ivan: use global user notification system to display the problem to the user as well
            Raven.captureException(error);
        }
    }
}

function* watchSaveTelemetryConfig() {
    yield takeLatest(types.SAVE_TELEMETRY_CONFIG, updateTelemetryConfig);
}

export default function* telemetryConfig() {
    yield all([
        takeEveryNewlyMatchedLocation(systemConfigPath, getTelemetryConfig),
        fork(watchSaveTelemetryConfig),
    ]);
}
