import { takeLatest, all, call, fork, put, select } from 'redux-saga/effects';
import { fetchGlobalSearchResults } from 'services/SearchService';
import { actions, types } from 'reducers/globalSearch';
import { actions as alertsActions } from 'reducers/alerts';
import { actions as imagesActions } from 'reducers/images';
import { actions as deploymentsActions } from 'reducers/deployments';
import { actions as environmentActions } from 'reducers/environment';
import { actions as secretsActions } from 'reducers/secrets';
import { selectors } from 'reducers';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

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
        case 'VIOLATIONS':
            yield put(alertsActions.setAlertsSearchOptions(searchOptions));
            break;
        case 'DEPLOYMENTS':
        case 'RISK':
            yield put(deploymentsActions.setDeploymentsSearchOptions(searchOptions));
            break;
        case 'SECRETS':
            yield put(secretsActions.setSecretsSearchOptions(searchOptions));
            break;
        case 'ENVIRONMENT':
            yield put(environmentActions.setEnvironmentSearchOptions(searchOptions));
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
        fork(watchPassthroughGlobalSearchOptions)
    ]);
}
