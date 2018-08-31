import { call, select } from 'redux-saga/effects';
import { expectSaga } from 'redux-saga-test-plan';
import { dynamic } from 'redux-saga-test-plan/providers';

import { selectors } from 'reducers';
import { actions, types } from 'reducers/secrets';
import { fetchSecrets } from 'services/SecretsService';
import saga from './secretSagas';
import createLocationChange from './sagaTestUtils';

describe('Secret Sagas', () => {
    const secretNameSearchOptions = [
        {
            value: 'Secret:',
            label: 'Secret:',
            type: 'categoryOption'
        },
        {
            value: 'bla',
            label: 'bla'
        }
    ];

    it('should get unfiltered list of secrets on Secrets page', () => {
        const secrets = ['secret1', 'secret2'];
        const fetchMock = jest.fn().mockReturnValue({ response: secrets });

        return expectSaga(saga)
            .provide([
                [select(selectors.getSecretsSearchOptions), []],
                [call(fetchSecrets, []), dynamic(fetchMock)]
            ])
            .dispatch(createLocationChange('/main/secrets'))
            .dispatch({ type: types.SET_SEARCH_OPTIONS, payload: { options: [] } })
            .put(actions.fetchSecrets.success(secrets, { options: [] }))
            .silentRun();
    });

    it('should get a filtered list of secrets on the secrets page', () => {
        const secrets = ['secret1', 'secret2'];
        const fetchMock = jest.fn().mockReturnValueOnce({ response: secrets });

        return expectSaga(saga)
            .provide([
                [select(selectors.getSecretsSearchOptions), secretNameSearchOptions],
                [call(fetchSecrets, secretNameSearchOptions), dynamic(fetchMock)]
            ])
            .put(actions.fetchSecrets.success(secrets, { options: secretNameSearchOptions }))
            .dispatch({
                type: types.SET_SEARCH_OPTIONS,
                payload: { options: secretNameSearchOptions }
            })
            .dispatch(createLocationChange('/main/secrets'))
            .silentRun();
    });

    it('should re-fetch secrets with new secrets search options', () => {
        const secrets = ['secret1', 'secret2'];
        const fetchMock = jest.fn().mockReturnValueOnce({ response: secrets });

        return expectSaga(saga)
            .provide([
                [select(selectors.getSecretsSearchOptions), secretNameSearchOptions],
                [call(fetchSecrets, secretNameSearchOptions), dynamic(fetchMock)]
            ])
            .put(actions.fetchSecrets.success(secrets, { options: secretNameSearchOptions }))
            .dispatch({
                type: types.SET_SEARCH_OPTIONS,
                payload: { options: secretNameSearchOptions }
            })
            .dispatch(actions.setSecretsSearchOptions(secretNameSearchOptions))
            .silentRun();
    });
});
