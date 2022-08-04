import { takeLatest, all, call, fork, put, select } from 'redux-saga/effects';
import { fetchGlobalSearchResults } from 'services/SearchService';
import { actions, types } from 'reducers/globalSearch';
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
        if (category !== 'SEARCH_UNSET') {
            filters.categories = [category];
        }
        const result = yield call(fetchGlobalSearchResults, filters);
        yield put(actions.fetchGlobalSearchResults.success(result, { category }));
    } catch (error) {
        yield put(actions.fetchGlobalSearchResults.failure(error));
        if (error.response && error.response.status >= 500 && error.response.data.error) {
            toast.error(error.response.data.error);
        }
    }
}

function* watchGlobalsearchSearchOptions() {
    yield takeLatest(types.SET_SEARCH_OPTIONS, getGlobalSearchResults);
}

function* watchSetGlobalSearchCategory() {
    yield takeLatest(types.SET_GLOBAL_SEARCH_CATEGORY, getGlobalSearchResults);
}

export default function* globalSearch() {
    yield all([fork(watchGlobalsearchSearchOptions), fork(watchSetGlobalSearchCategory)]);
}
