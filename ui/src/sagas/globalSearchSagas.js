import { takeLatest, all, call, fork, put, select } from 'redux-saga/effects';

import { mainPath } from 'routePaths';
import { fetchGlobalSearchResults } from 'services/SearchService';
import { actions, types } from 'reducers/globalSearch';
import { actions as alertsActions } from 'reducers/alerts';
import { actions as imagesActions } from 'reducers/images';
import { actions as deploymentsActions } from 'reducers/deployments';
import { selectors } from 'reducers';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import { setStaleSearchOption } from 'utils/searchUtils';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';

import { toast } from 'react-toastify';

export function* getGlobalSearchResults() {
    try {
        const searchOptions = yield select(selectors.getGlobalSearchOptions);
        const category = yield select(selectors.getGlobalSearchCategory);
        const filters = {
            query: searchOptionsToQuery(searchOptions)
        };
        if (category !== '') filters.categories = category;
        const result = yield call(fetchGlobalSearchResults, filters);
        yield put(actions.fetchGlobalSearchResults.success(result.response));
    } catch (error) {
        yield put(actions.fetchGlobalSearchResults.failure(error));
        if (error.response && error.response.status >= 500 && error.response.data.error)
            toast.error(error.response.data.error);
    }
}

export function* passthroughGlobalSearchOptions({ searchOptions, category }) {
    switch (category) {
        case 'IMAGES':
            yield put(imagesActions.setImagesSearchOptions(searchOptions));
            break;
        case 'ALERTS':
            yield put(alertsActions.setAlertsSearchOptions(searchOptions));
            break;
        case 'DEPLOYMENTS':
            yield put(deploymentsActions.setDeploymentsSearchOptions(searchOptions));
            break;
        default:
            break;
    }
}

function* setStaleSearchOptionInGlobalSearch() {
    let searchOptions = yield select(selectors.getGlobalSearchOptions);
    searchOptions = setStaleSearchOption(searchOptions);
    yield put(actions.setGlobalSearchOptions(searchOptions));
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
        takeEveryNewlyMatchedLocation(mainPath, setStaleSearchOptionInGlobalSearch),
        fork(watchGlobalsearchSearchOptions),
        fork(watchSetGlobalSearchCategory),
        fork(watchPassthroughGlobalSearchOptions)
    ]);
}
