import { take, fork, put, call } from 'redux-saga/effects';

import { actions as alertActions } from 'reducers/alerts';
import { actions as riskActions } from 'reducers/risk';
import { actions as policiesActions } from 'reducers/policies';
import { actions as imagesActions } from 'reducers/images';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { types as locationActionTypes } from 'reducers/routes';
import { fetchOptions } from 'services/SearchService';

const violationsPath = '/main/violations';
const riskPath = '/main/risk';
const policiesPath = '/main/policies';
const imagesPath = '/main/images';
const mainPath = '/main';

export function* getSearchOptions(setSearchModifiers, setSearchSuggestions, query = '') {
    try {
        const result = yield call(fetchOptions, query);
        yield put(setSearchModifiers(result.options));
        yield put(setSearchSuggestions(result.options));
    } catch (error) {
        yield put(setSearchModifiers([]));
        yield put(setSearchSuggestions([]));
    }
}

export function* watchLocation() {
    let globalSearchDone = false;
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

        if (
            location &&
            location.pathname &&
            location.pathname.startsWith(mainPath) &&
            !globalSearchDone
        ) {
            yield fork(
                getSearchOptions,
                globalSearchActions.setGlobalSearchModifiers,
                globalSearchActions.setGlobalSearchSuggestions
            );
            globalSearchDone = true;
        }

        if (location && location.pathname && location.pathname.startsWith(violationsPath)) {
            yield fork(
                getSearchOptions,
                alertActions.setAlertsSearchModifiers,
                alertActions.setAlertsSearchSuggestions,
                'categories=ALERTS'
            );
        } else if (location && location.pathname && location.pathname.startsWith(riskPath)) {
            yield fork(
                getSearchOptions,
                riskActions.setDeploymentsSearchModifiers,
                riskActions.setDeploymentsSearchSuggestions,
                'categories=DEPLOYMENTS'
            );
        } else if (location && location.pathname && location.pathname.startsWith(policiesPath)) {
            yield fork(
                getSearchOptions,
                policiesActions.setPoliciesSearchModifiers,
                policiesActions.setPoliciesSearchSuggestions,
                'categories=POLICIES'
            );
        } else if (location && location.pathname && location.pathname.startsWith(imagesPath)) {
            yield fork(
                getSearchOptions,
                imagesActions.setImagesSearchModifiers,
                imagesActions.setImagesSearchSuggestions,
                'categories=IMAGES'
            );
        }
    }
}

export default function* searches() {
    yield fork(watchLocation);
}
