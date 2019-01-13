import { takeLatest, call, fork, put, all } from 'redux-saga/effects';
import Raven from 'raven-js';

import downloadCLI from 'services/CLIService';
import { actions as notificationActions } from 'reducers/notifications';
import { types } from 'reducers/cli';

function* downloadCLIFile({ os }) {
    try {
        yield call(downloadCLI, os);
    } catch (error) {
        yield put(notificationActions.addNotification('Error while downloading a file'));
        yield put(notificationActions.removeOldestNotification());
        Raven.captureException(error);
    }
}

function* watchDownloadCLI() {
    yield takeLatest(types.CLI_DOWNLOAD, downloadCLIFile);
}

export default function* cli() {
    yield all([fork(watchDownloadCLI)]);
}
