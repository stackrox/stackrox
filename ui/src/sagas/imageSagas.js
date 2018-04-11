import { all, take, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import fetchImages from 'services/ImagesService';
import { actions, types } from 'reducers/images';
import { types as locationActionTypes } from 'reducers/routes';
import { selectors } from 'reducers';

const imagesPath = '/main/images';

export function* getImages() {
    try {
        const searchQuery = yield select(selectors.getImagesSearchQuery);
        const filters = {
            query: searchQuery
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

export function* watchLocation() {
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (location && location.pathname && location.pathname.startsWith(imagesPath)) {
            yield fork(getImages);
        }
    }
}

export default function* images() {
    yield all([fork(watchLocation), fork(watchImagesSearchOptions)]);
}
