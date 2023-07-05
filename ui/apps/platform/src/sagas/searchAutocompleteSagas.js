import { take, call, fork, put } from 'redux-saga/effects';

import * as service from 'services/SearchService';
import { actions, types } from 'reducers/searchAutocomplete';

function* getAutoCompleteResults(request) {
    try {
        const result = yield call(service.fetchAutoCompleteResults, request);
        yield put(actions.recordAutoCompleteResponse(result));
    } catch (error) {
        yield put(actions.recordAutoCompleteResponse([]));
    }
}

function* watchFetchRequest() {
    while (true) {
        const request = yield take(types.SEND_AUTOCOMPLETE_REQUEST);
        yield fork(getAutoCompleteResults, request);
    }
}

export default function* searchAutoComplete() {
    yield fork(watchFetchRequest);
}
