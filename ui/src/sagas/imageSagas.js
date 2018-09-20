import { all, takeLatest, call, fork, put, select } from 'redux-saga/effects';

import { imagesPath } from 'routePaths';
import { fetchImages, fetchImage } from 'services/ImagesService';
import { actions, types } from 'reducers/images';
import { selectors } from 'reducers';
import { takeEveryNewlyMatchedLocation, takeEveryLocation } from 'utils/sagaEffects';

function* getImages({ options = [] }) {
    try {
        const result = yield call(fetchImages, options);
        yield put(actions.fetchImages.success(result.response, { options }));
    } catch (error) {
        yield put(actions.fetchImages.failure(error));
    }
}

export function* getImage(sha) {
    try {
        yield put(actions.fetchImage.request());
        const result = yield call(fetchImage, sha);
        yield put(actions.fetchImage.success(result.response));
    } catch (error) {
        yield put(actions.fetchImage.failure(error));
    }
}

function* filterImagesPageBySearch() {
    const options = yield select(selectors.getImagesSearchOptions);
    if (options.length && options[options.length - 1].type) {
        return;
    }
    yield fork(getImages, { options });
}

function* watchImagesSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, filterImagesPageBySearch);
}

function* getSelectedImage({ match }) {
    const { imageSha } = match.params;
    if (imageSha) {
        // grpc does not take ':' so we are passing the hash after to the server
        yield fork(getImage, imageSha.split(':')[1]);
    }
}

export default function* images() {
    yield all([
        takeEveryNewlyMatchedLocation(imagesPath, filterImagesPageBySearch),
        takeEveryLocation(imagesPath, getSelectedImage),
        fork(watchImagesSearchOptions)
    ]);
}
