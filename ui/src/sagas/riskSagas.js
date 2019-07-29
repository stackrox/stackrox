import { all, call, put } from 'redux-saga/effects';
import { riskPath } from 'routePaths';
import { takeEveryLocation } from 'utils/sagaEffects';
import fetchRisk from 'services/RisksService';
import { actions } from 'reducers/risks';

import Raven from 'raven-js';

export function* getRisk(subjectId, subjectType) {
    try {
        const result = yield call(fetchRisk, subjectId, subjectType);
        yield put(actions.fetchRisk.success(result.response));
    } catch (error) {
        yield put(actions.fetchRisk.failure(error));
        Raven.captureException(error);
    }
}

function* getRiskByDeployment({ match }) {
    const { deploymentId } = match.params;
    if (deploymentId) {
        try {
            yield put(actions.fetchRisk.request());
            yield call(getRisk, deploymentId, 'deployment');
        } catch (error) {
            Raven.captureException(error);
        }
    }
}

export default function* integrations() {
    yield all([takeEveryLocation(riskPath, getRiskByDeployment)]);
}
