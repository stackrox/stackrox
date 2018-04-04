import { take, fork, put, call } from 'redux-saga/effects';

import { actions as alertActions } from 'reducers/alerts';
import { actions as riskActions } from 'reducers/risk';
import { actions as policiesActions } from 'reducers/policies';
import { types as locationActionTypes } from 'reducers/routes';
import fetchOptions from 'services/SearchService';

const violationsPath = '/main/violations';
const riskPath = '/main/risk';
const policiesPath = '/main/policies';

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
    while (true) {
        const action = yield take(locationActionTypes.LOCATION_CHANGE);
        const { payload: location } = action;

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
        }
    }
}

export default function* searches() {
    yield fork(watchLocation);
}
