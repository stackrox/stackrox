import { delay } from 'redux-saga';
import { all, take, takeLatest, call, fork, put, cancel, select } from 'redux-saga/effects';

import fetchImages from 'services/ImagesService';
import { actions, types } from 'reducers/images';
import { types as locationActionTypes } from 'reducers/routes';
import { selectors } from 'reducers';

const imagesPath = '/main/images';

export function* getImages() {
    while (true) {
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
        yield delay(5000);
    }
}

function* watchImagesSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, getImages);
}

export function* watchLocation() {
    let pollTask;
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (pollTask) yield cancel(pollTask); // cancel polling in any case
        if (location && location.pathname && location.pathname.startsWith(imagesPath)) {
            pollTask = yield fork(getImages);
        }
    }
}

export default function* images() {
    yield all([fork(watchLocation), fork(watchImagesSearchOptions)]);
}
