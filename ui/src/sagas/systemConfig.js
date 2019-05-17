import { all, takeLatest, call, fork, put } from 'redux-saga/effects';
import Raven from 'raven-js';

import { mainPath, loginPath, systemConfigPath } from 'routePaths';
import {
    fetchSystemConfig,
    fetchPublicConfig,
    saveSystemConfig
} from 'services/SystemConfigService';
import { actions, types } from 'reducers/systemConfig';
import { actions as notificationActions } from 'reducers/notifications';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* getSystemConfig() {
    try {
        const { response } = yield call(fetchSystemConfig);
        yield put(actions.fetchSystemConfig.success(response));
    } catch (error) {
        yield put(actions.fetchSystemConfig.failure(error));
    }
}

function* getPublicConfig() {
    try {
        const { response } = yield call(fetchPublicConfig);
        yield put(actions.fetchPublicConfig.success(response));
    } catch (error) {
        yield put(actions.fetchPublicConfig.failure(error));
    }
}

function* updateSystemConfig(action) {
    try {
        const config = { config: action.systemConfig };
        yield call(saveSystemConfig, config);
        yield fork(getPublicConfig);
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

function* watchSaveSystemConfig() {
    yield takeLatest(types.SAVE_SYSTEM_CONFIG, updateSystemConfig);
}

export default function* systemConfig() {
    yield all([
        takeEveryNewlyMatchedLocation(systemConfigPath, getSystemConfig),
        takeEveryNewlyMatchedLocation(loginPath, getPublicConfig),
        takeEveryNewlyMatchedLocation(mainPath, getPublicConfig),
        fork(watchSaveSystemConfig)
    ]);
}
