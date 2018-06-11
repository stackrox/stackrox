import { all, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import { imagesPath } from 'routePaths';
import fetchImages from 'services/ImagesService';
import { actions, types } from 'reducers/images';
import { selectors } from 'reducers';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

function* getImages() {
    try {
        const searchOptions = yield select(selectors.getImagesSearchOptions);
        const filters = {
            query: searchOptionsToQuery(searchOptions)
        };
        const result = yield call(fetchImages, filters);
        yield put(actions.fetchImages.success(result.response));
    } catch (error) {
        yield put(actions.fetchImages.failure(error));
    }
}

function* watchImagesSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, getImages);
}

export default function* images() {
    yield all([
        takeEveryNewlyMatchedLocation(imagesPath, getImages),
        fork(watchImagesSearchOptions)
    ]);
}
