import { take, call, fork, put, select } from 'redux-saga/effects';

import * as service from 'services/SearchService';
import { actions, types } from 'reducers/searchAutocomplete';
import { selectors } from 'reducers';

function* getAutoCompleteResults(request) {
    try {
        const result = yield call(service.fetchAutoCompleteResults, request);
        yield put(actions.recordAutoCompleteResponse(result));
    } catch (error) {
        yield put(actions.recordAutoCompleteResponse([]));
    }
}

function* getNetworkAutoCompleteResults(request) {
    const selectedClusterId = yield select(selectors.getSelectedNetworkClusterId);
    const queryWithClusterId = `Cluster Id:${selectedClusterId}+${request.query}`;
    const newRequest = { ...request, query: queryWithClusterId };
    yield getAutoCompleteResults(newRequest);
}

function* watchFetchRequest() {
    while (true) {
        const request = yield take(types.SEND_AUTOCOMPLETE_REQUEST);
        const location = yield select(selectors.getLocation);
        if (location.pathname === '/main/network') {
            yield fork(getNetworkAutoCompleteResults, request);
        } else {
            yield fork(getAutoCompleteResults, request);
        }
    }
}

export default function* searchAutoComplete() {
    yield fork(watchFetchRequest);
}
