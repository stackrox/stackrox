import { takeLatest, all, call, fork, put, select } from 'redux-saga/effects';
import { fetchGlobalSearchResults } from 'services/SearchService';
import { actions, types } from 'reducers/globalSearch';
import { actions as policiesActions } from 'reducers/policies/search';
import { selectors } from 'reducers';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

import { toast } from 'react-toastify';

export function* getGlobalSearchResults() {
    try {
        const searchOptions = yield select(selectors.getGlobalSearchOptions);
        if (searchOptions.length && searchOptions[searchOptions.length - 1].type) {
            return;
        }
        const category = yield select(selectors.getGlobalSearchCategory);
        const filters = {
            query: searchOptionsToQuery(searchOptions),
        };
        if (category !== '') {
            filters.categories = category;
        }
        const result = yield call(fetchGlobalSearchResults, filters);
        yield put(actions.fetchGlobalSearchResults.success(result.response, { category }));
    } catch (error) {
        yield put(actions.fetchGlobalSearchResults.failure(error));
        if (error.response && error.response.status >= 500 && error.response.data.error) {
            toast.error(error.response.data.error);
        }
    }
}

export function* passthroughGlobalSearchOptions({ searchOptions, category }) {
    switch (category) {
        case 'POLICIES':
            yield put(policiesActions.setPoliciesSearchOptions(searchOptions));
            break;
        default:
            break;
    }
}

function* watchGlobalsearchSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, getGlobalSearchResults);
}

function* watchSetGlobalSearchCategory() {
    yield takeLatest(types.SET_GLOBAL_SEARCH_CATEGORY, getGlobalSearchResults);
}

function* watchPassthroughGlobalSearchOptions() {
    yield takeLatest(types.PASSTHROUGH_GLOBAL_SEARCH_OPTIONS, passthroughGlobalSearchOptions);
}

export default function* globalSearch() {
    yield all([
        fork(watchGlobalsearchSearchOptions),
        fork(watchSetGlobalSearchCategory),
        fork(watchPassthroughGlobalSearchOptions),
    ]);
}
