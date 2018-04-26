import { take, fork, put, call } from 'redux-saga/effects';
import { types as locationActionTypes } from 'reducers/routes';
import { actions as alertActions } from 'reducers/alerts';
import { actions as riskActions } from 'reducers/risk';
import { actions as policiesActions } from 'reducers/policies';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { fetchOptions } from 'services/SearchService';
import { getSearchOptions, watchLocation } from './searchSagas';

describe('Search Sagas Test', () => {
    it('Should load search modifiers/suggestions for Global Search when location changes', () => {
        const gen = watchLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/violations'
            }
        }));
        expect(value).toEqual(
            fork(
                getSearchOptions,
                globalSearchActions.setGlobalSearchModifiers,
                globalSearchActions.setGlobalSearchSuggestions
            )
        );
    });
    it('Should load search modifiers/suggestions for Alerts when location changes to Violations Page', () => {
        const gen = watchLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/violations'
            }
        }));
        expect(value).toEqual(
            fork(
                getSearchOptions,
                globalSearchActions.setGlobalSearchModifiers,
                globalSearchActions.setGlobalSearchSuggestions
            )
        );
        ({ value } = gen.next());
        expect(value).toEqual(
            fork(
                getSearchOptions,
                alertActions.setAlertsSearchModifiers,
                alertActions.setAlertsSearchSuggestions,
                alertActions.setAlertsSearchOptions,
                'categories=ALERTS'
            )
        );
    });
    it('Should load search modifiers/suggestions for Deployments when location changes to Risk Page', () => {
        const gen = watchLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/risk'
            }
        }));
        expect(value).toEqual(
            fork(
                getSearchOptions,
                globalSearchActions.setGlobalSearchModifiers,
                globalSearchActions.setGlobalSearchSuggestions
            )
        );
        ({ value } = gen.next());
        expect(value).toEqual(
            fork(
                getSearchOptions,
                riskActions.setDeploymentsSearchModifiers,
                riskActions.setDeploymentsSearchSuggestions,
                riskActions.setDeploymentsSearchOptions,
                'categories=DEPLOYMENTS'
            )
        );
    });
    it('Should load search modifiers/suggestions for Policies when location changes to Policies Page', () => {
        const gen = watchLocation();
        let { value } = gen.next();
        expect(value).toEqual(take(locationActionTypes.LOCATION_CHANGE));
        ({ value } = gen.next({
            type: locationActionTypes.LOCATION_CHANGE,
            payload: {
                pathname: '/main/policies'
            }
        }));
        expect(value).toEqual(
            fork(
                getSearchOptions,
                globalSearchActions.setGlobalSearchModifiers,
                globalSearchActions.setGlobalSearchSuggestions
            )
        );
        ({ value } = gen.next());
        expect(value).toEqual(
            fork(
                getSearchOptions,
                policiesActions.setPoliciesSearchModifiers,
                policiesActions.setPoliciesSearchSuggestions,
                policiesActions.setPoliciesSearchOptions,
                'categories=POLICIES'
            )
        );
    });
    it('Should set searchModifiers, and searchSuggestions when getSearchOptions is called', () => {
        const result = {
            options: ['OPTION 1', 'OPTION 2', 'OPTION 3']
        };
        const setSearchModifiers = alertActions.setAlertsSearchModifiers;
        const setSearchSuggestions = alertActions.setAlertsSearchSuggestions;
        const setSearchOptions = alertActions.setAlertsSearchOptions;
        const gen = getSearchOptions(
            setSearchModifiers,
            setSearchSuggestions,
            setSearchOptions,
            'categories=TEST'
        );
        let { value } = gen.next();
        expect(value).toEqual(call(fetchOptions, 'categories=TEST'));
        ({ value } = gen.next(result));
        expect(value).toEqual(put(setSearchModifiers(result.options)));
        ({ value } = gen.next());
        expect(value).toEqual(put(setSearchSuggestions(result.options)));
    });
});
