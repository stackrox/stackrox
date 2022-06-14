import { all, put, call } from 'redux-saga/effects';

import { mainPath, policiesPath } from 'routePaths';
import { takeEveryNewlyMatchedLocation } from 'utils/sagaEffects';
import { actions as policiesActions } from 'reducers/policies/search';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { fetchOptions } from 'services/SearchService';
import capitalize from 'lodash/capitalize';

const getParams = (url) => {
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
        keys.forEach((key) => {
            queryOptions.push(
                {
                    label: `${capitalize(key)}:`,
                    type: 'categoryOption',
                    value: `${capitalize(key)}:`,
                },
                {
                    className: 'Select-create-option-placeholder',
                    label: searchParams[key],
                    value: searchParams[key],
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
        if (queryOptions.length && setSearchOptions) {
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
            globalSearchActions.setGlobalSearchSuggestions,
            null,
            ''
        ),
        // TODO: remove once policies is fully migrated over to PF
        takeEveryNewlyMatchedLocation(
            policiesPath,
            getSearchOptions,
            policiesActions.setPoliciesSearchModifiers,
            policiesActions.setPoliciesSearchSuggestions,
            policiesActions.setPoliciesSearchOptions,
            'categories=POLICIES'
        ),
    ]);
}
