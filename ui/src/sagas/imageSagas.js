import { delay } from 'redux-saga';
import { take, call, fork, put, cancel } from 'redux-saga/effects';

import fetchImages from 'services/ImagesService';
import { actions } from 'reducers/images';
import { types as locationActionTypes } from 'reducers/routes';

const imagesPath = '/main/images';

export function* getImages() {
    while (true) {
        try {
            const result = yield call(fetchImages);
            yield put(actions.fetchImages.success(result.response));
        } catch (error) {
            yield put(actions.fetchImages.failure(error));
        }
        yield delay(5000);
    }
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
    yield fork(watchLocation);
}
