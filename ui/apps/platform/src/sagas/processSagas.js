import { all, call, fork, put, takeLatest } from 'redux-saga/effects';
import { riskPath } from 'routePaths';
import {
    fetchProcesses,
    fetchProcessesInBaseline,
    addDeleteProcesses,
    lockUnlockProcesses,
} from 'services/ProcessesService';
import { actions, types } from 'reducers/processes';
import { getDeploymentAndProcessIdFromGroupedProcesses } from 'utils/processUtils';
import { takeEveryLocation } from 'utils/sagaEffects';
import Raven from 'raven-js';
import uniqBy from 'lodash/uniqBy';

export function* getProcesses(id) {
    try {
        const result = yield call(fetchProcesses, id);
        yield put(actions.fetchProcesses.success(result.response));
        const promises = [];
        const uniqueContainerNames = uniqBy(result.response.groups, 'containerName').map(
            (x) => x.containerName
        );

        const { clusterId, namespace } = getDeploymentAndProcessIdFromGroupedProcesses(
            result.response.groups
        );

        if (clusterId && namespace && uniqueContainerNames && uniqueContainerNames.length) {
            uniqueContainerNames.forEach((containerName) => {
                const queryStr = `key.clusterId=${clusterId}&key.namespace=${namespace}&key.deploymentId=${id}&key.containerName=${containerName}`;
                promises.push(call(fetchProcessesInBaseline, queryStr));
            });
        }

        const processesBaseline = yield all(promises);
        yield put(actions.fetchProcessesBaseline.success(processesBaseline));
    } catch (error) {
        yield put(actions.fetchProcesses.failure(error));
        yield put(actions.fetchProcessesBaseline.failure(error));
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

function* addDeleteProcessesBaseline(action) {
    try {
        const { deploymentId } = action.processes.keys[0];
        yield call(addDeleteProcesses, action.processes);
        yield call(getProcesses, deploymentId);
    } catch (error) {
        Raven.captureException(error);
    }
}

function* lockUnlockProcessesBaseline(action) {
    try {
        const { deploymentId } = action.processes.keys[0];
        yield call(lockUnlockProcesses, action.processes);
        yield call(getProcesses, deploymentId);
    } catch (error) {
        Raven.captureException(error);
    }
}

function* watchAddDeleteProcesses() {
    yield takeLatest(types.ADD_DELETE_PROCESSES, addDeleteProcessesBaseline);
}

function* watchLockUnlockProcesses() {
    yield takeLatest(types.LOCK_UNLOCK_PROCESSES, lockUnlockProcessesBaseline);
}

export default function* processes() {
    yield all([
        takeEveryLocation(riskPath, getProcessesByDeployment),
        fork(watchAddDeleteProcesses),
        fork(watchLockUnlockProcesses),
    ]);
}
