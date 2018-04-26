import { take, fork, put, call } from 'redux-saga/effects';

import { actions as alertActions } from 'reducers/alerts';
import { actions as riskActions } from 'reducers/risk';
import { actions as policiesActions } from 'reducers/policies';
import { actions as imagesActions } from 'reducers/images';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { types as locationActionTypes } from 'reducers/routes';
import { fetchOptions } from 'services/SearchService';
import capitalize from 'lodash/capitalize';

const violationsPath = '/main/violations';
const riskPath = '/main/risk';
const policiesPath = '/main/policies';
const imagesPath = '/main/images';
const mainPath = '/main';

const getParams = url => {
    const params = {};
    const parser = document.createElement('a');
    parser.href = url;
    const query = parser.search.substring(1);
    const vars = query.split('&');
    for (let i = 0; i < vars.length; i += 1) {
        const pair = vars[i].split('=');
        if (pair[0] !== '') {
            params[pair[0]] = decodeURIComponent(pair[1]);
        }
    }
    return params;
};

const getQuery = () => {
    const searchParams = getParams(window.location.href);
    const keys = Object.keys(searchParams);
    const queryOptions = [];

    if (keys.length) {
        keys.forEach(key => {
            queryOptions.push(
                {
                    label: `${capitalize(key)}:`,
                    type: 'categoryOption',
                    value: `${capitalize(key)}:`
                },
                {
                    className: 'Select-create-option-placeholder',
                    label: searchParams[key],
                    value: searchParams[key]
                }
            );
        });
    }

    return queryOptions;
};

export function* getSearchOptions(
    setSearchModifiers,
    setSearchSuggestions,
    setSearchOptions,
    query = ''
) {
    try {
        const result = yield call(fetchOptions, query);
        yield put(setSearchModifiers(result.options));
        yield put(setSearchSuggestions(result.options));
        const queryOptions = getQuery();
        if (queryOptions.length) {
            yield put(setSearchOptions(queryOptions));
        }
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
                alertActions.setAlertsSearchOptions,
                `categories=ALERTS`
            );
        } else if (location && location.pathname && location.pathname.startsWith(riskPath)) {
            yield fork(
                getSearchOptions,
                riskActions.setDeploymentsSearchModifiers,
                riskActions.setDeploymentsSearchSuggestions,
                riskActions.setDeploymentsSearchOptions,
                'categories=DEPLOYMENTS'
            );
        } else if (location && location.pathname && location.pathname.startsWith(policiesPath)) {
            yield fork(
                getSearchOptions,
                policiesActions.setPoliciesSearchModifiers,
                policiesActions.setPoliciesSearchSuggestions,
                policiesActions.setPoliciesSearchOptions,
                'categories=POLICIES'
            );
        } else if (location && location.pathname && location.pathname.startsWith(imagesPath)) {
            yield fork(
                getSearchOptions,
                imagesActions.setImagesSearchModifiers,
                imagesActions.setImagesSearchSuggestions,
                imagesActions.setImagesSearchOptions,
                'categories=IMAGES'
            );
        }
    }
}

export default function* searches() {
    yield fork(watchLocation);
}
