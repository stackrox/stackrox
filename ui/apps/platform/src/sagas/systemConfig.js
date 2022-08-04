import { all, call, put } from 'redux-saga/effects';

import { mainPath, loginPath } from 'routePaths';
import { fetchPublicConfig } from 'services/SystemConfigService';
import { actions } from 'reducers/systemConfig';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* getPublicConfig() {
    try {
        const { response } = yield call(fetchPublicConfig);
        yield put(actions.fetchPublicConfig.success(response));
    } catch (error) {
        yield put(actions.fetchPublicConfig.failure(error));
    }
}

export default function* systemConfig() {
    yield all([
        takeEveryNewlyMatchedLocation(loginPath, getPublicConfig),
        takeEveryNewlyMatchedLocation(mainPath, getPublicConfig),
    ]);
}
