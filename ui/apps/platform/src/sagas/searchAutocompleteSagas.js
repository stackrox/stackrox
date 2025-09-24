import { call, fork, put, take } from 'redux-saga/effects';

import { fetchAutoCompleteResults as serviceFetchAutoCompleteResults } from 'services/SearchService';
import { actions, types } from 'reducers/searchAutocomplete';

function* getAutoCompleteResults(request) {
    try {
        const result = yield call(serviceFetchAutoCompleteResults, request);
        yield put(actions.recordAutoCompleteResponse(result));
    } catch {
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
