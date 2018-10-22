import { all, put, call } from 'redux-saga/effects';

import {
    mainPath,
    dashboardPath,
    violationsPath,
    riskPath,
    policiesPath,
    imagesPath,
    secretsPath,
    networkPath
} from 'routePaths';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import { actions as alertActions } from 'reducers/alerts';
import { actions as deploymentsActions } from 'reducers/deployments';
import { actions as policiesActions } from 'reducers/policies/search';
import { actions as imagesActions } from 'reducers/images';
import { actions as secretsActions } from 'reducers/secrets';
import { actions as dashboardActions } from 'reducers/dashboard';
import { actions as networkActions } from 'reducers/network';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { fetchOptions } from 'services/SearchService';
import capitalize from 'lodash/capitalize';

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

function* getSearchOptions(setSearchModifiers, setSearchSuggestions, setSearchOptions, query = '') {
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

export default function* searches() {
    yield all([
        takeEveryNewlyMatchedLocation(
            mainPath,
            getSearchOptions,
            globalSearchActions.setGlobalSearchModifiers,
            globalSearchActions.setGlobalSearchSuggestions
        ),
        takeEveryNewlyMatchedLocation(
            violationsPath,
            getSearchOptions,
            alertActions.setAlertsSearchModifiers,
            alertActions.setAlertsSearchSuggestions,
            alertActions.setAlertsSearchOptions,
            `categories=ALERTS`
        ),
        takeEveryNewlyMatchedLocation(
            riskPath,
            getSearchOptions,
            deploymentsActions.setDeploymentsSearchModifiers,
            deploymentsActions.setDeploymentsSearchSuggestions,
            deploymentsActions.setDeploymentsSearchOptions,
            'categories=DEPLOYMENTS'
        ),
        takeEveryNewlyMatchedLocation(
            policiesPath,
            getSearchOptions,
            policiesActions.setPoliciesSearchModifiers,
            policiesActions.setPoliciesSearchSuggestions,
            policiesActions.setPoliciesSearchOptions,
            'categories=POLICIES'
        ),
        takeEveryNewlyMatchedLocation(
            imagesPath,
            getSearchOptions,
            imagesActions.setImagesSearchModifiers,
            imagesActions.setImagesSearchSuggestions,
            imagesActions.setImagesSearchOptions,
            'categories=IMAGES'
        ),
        takeEveryNewlyMatchedLocation(
            secretsPath,
            getSearchOptions,
            secretsActions.setSecretsSearchModifiers,
            secretsActions.setSecretsSearchSuggestions,
            secretsActions.setSecretsSearchOptions,
            'categories=SECRETS'
        ),
        takeEveryNewlyMatchedLocation(
            dashboardPath,
            getSearchOptions,
            dashboardActions.setDashboardSearchModifiers,
            dashboardActions.setDashboardSearchSuggestions,
            dashboardActions.setDashboardSearchOptions,
            null
        ),
        takeEveryNewlyMatchedLocation(
            networkPath,
            getSearchOptions,
            networkActions.setNetworkSearchModifiers,
            networkActions.setNetworkSearchSuggestions,
            networkActions.setNetworkSearchOptions,
            'categories=DEPLOYMENTS'
        )
    ]);
}
