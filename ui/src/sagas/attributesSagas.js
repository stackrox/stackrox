import { all, call, put } from 'redux-saga/effects';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import { accessControlPath } from 'routePaths';
import fetchUsersAttributes from 'services/AttributesService';
import { actions } from 'reducers/attributes';

function* getUsersAttributes() {
    try {
        const result = yield call(fetchUsersAttributes);
        yield put(actions.fetchUsersAttributes.success(result.response));
    } catch (error) {
        yield put(actions.fetchUsersAttributes.failure(error));
    }
}

export default function* groups() {
    yield all([takeEveryNewlyMatchedLocation(accessControlPath, getUsersAttributes)]);
}
