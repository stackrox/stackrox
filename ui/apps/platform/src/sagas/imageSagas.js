import { call, fork, put, select } from 'redux-saga/effects';

import { policiesPath } from 'routePaths';
import { fetchImagesById } from 'services/ImagesService';
import { actions } from 'reducers/images';
import { selectors } from 'reducers';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* getImages({ options = [] }) {
    try {
        const result = yield call(fetchImagesById, options);
        yield put(actions.fetchImages.success(result.response, { options }));
    } catch (error) {
        yield put(actions.fetchImages.failure(error));
    }
}

function* filterPoliciesPageBySearch() {
    const options = yield select(selectors.getPoliciesSearchOptions);
    if (options.length && options[options.length - 1].type) {
        return;
    }
    yield fork(getImages, { options });
}

export default function* images() {
    yield takeEveryNewlyMatchedLocation(policiesPath, filterPoliciesPageBySearch);
}
