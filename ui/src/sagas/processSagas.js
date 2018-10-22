import { all, call, put } from 'redux-saga/effects';
import { riskPath } from 'routePaths';
import fetchProcesses from 'services/ProcessesService';
import { actions } from 'reducers/processes';
import { takeEveryLocation } from 'utils/sagaEffects';
import Raven from 'raven-js';

export function* getProcesses(id) {
    try {
        const result = yield call(fetchProcesses, id);
        yield put(actions.fetchProcesses.success(result.response));
    } catch (error) {
        yield put(actions.fetchProcesses.failure(error));
        Raven.captureException(error);
    }
}

function* getProcessesByDeployment({ match }) {
    const { deploymentId } = match.params;
    if (deploymentId) {
        try {
            yield put(actions.fetchProcesses.request());
            yield call(getProcesses, deploymentId);
        } catch (error) {
            Raven.captureException(error);
        }
    }
}

export default function* processes() {
    yield all([takeEveryLocation(riskPath, getProcessesByDeployment)]);
}
