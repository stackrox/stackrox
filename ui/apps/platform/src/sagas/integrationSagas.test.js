import { call } from 'redux-saga/effects';
import { expectSaga } from 'redux-saga-test-plan';
import { dynamic } from 'redux-saga-test-plan/providers';

import { fetchIntegration } from 'services/IntegrationsService';
import { actions } from 'reducers/integrations';
import createLocationChange from './sagaTestUtils';
import saga from './integrationSagas';

describe('Integrations Sagas', () => {
    it('should fetch image integrations and notifiers when location changes to integrations', () => {
        const authPlugin = { authPlugin: { endpoint: 'endpoint' } };
        const imageIntegrations = { integrations: ['int1'] };
        const notifiers = { notifiers: ['notifier1'] };
        const backups = { backups: ['backup1'] };
        const signatureIntegrations = { signatureIntegrations: ['signatureIntegration1'] };
        return expectSaga(saga)
            .provide([
                [call(fetchIntegration, 'authPlugins'), { response: authPlugin }],
                [call(fetchIntegration, 'imageIntegrations'), { response: imageIntegrations }],
                [
                    call(fetchIntegration, 'signatureIntegrations'),
                    { response: signatureIntegrations },
                ],
                [call(fetchIntegration, 'backups'), { response: backups }],
                [call(fetchIntegration, 'notifiers'), { response: notifiers }],
            ])
            .put(actions.fetchImageIntegrations.success(imageIntegrations))
            .put(actions.fetchNotifiers.success(notifiers))
            .dispatch(createLocationChange('/main/integrations'))
            .silentRun();
    });

    it('should fetch notifiers for policies page once', () => {
        const notifiers = { notifiers: ['notifier1'] };
        const fetchMock = jest.fn().mockReturnValueOnce({ response: notifiers });

        return expectSaga(saga)
            .provide([[call(fetchIntegration, 'notifiers'), dynamic(fetchMock)]])
            .put(actions.fetchNotifiers.success(notifiers))
            .dispatch(createLocationChange('/main/policies'))
            .dispatch(createLocationChange('/main/policies/123'))
            .dispatch(createLocationChange('/main/policies/321'))
            .silentRun()
            .then(() => {
                expect(fetchMock.mock.calls.length).toBe(1);
            });
    });

    it("shouldn't fetch image integrations / notifiers when location changes to violations, dashboard, etc.", () => {
        const fetchImageIntegrationsMock = jest.fn();
        const fetchNotifiersMock = jest.fn();
        const fetchBackupsMock = jest.fn();
        const fetchAuthPluginMock = jest.fn();
        return expectSaga(saga)
            .provide([
                [call(fetchIntegration, 'authPlugins'), dynamic(fetchBackupsMock)],
                [call(fetchIntegration, 'imageIntegrations'), dynamic(fetchImageIntegrationsMock)],
                [call(fetchIntegration, 'notifiers'), dynamic(fetchNotifiersMock)],
                [call(fetchIntegration, 'backups'), dynamic(fetchBackupsMock)],
            ])
            .dispatch(createLocationChange('/main/violations'))
            .dispatch(createLocationChange('/main/compliance'))
            .dispatch(createLocationChange('/main/dashboard'))
            .silentRun()
            .then(() => {
                expect(fetchImageIntegrationsMock.mock.calls.length).toBe(0);
                expect(fetchNotifiersMock.mock.calls.length).toBe(0);
                expect(fetchBackupsMock.mock.calls.length).toBe(0);
                expect(fetchAuthPluginMock.mock.calls.length).toBe(0);
            });
    });
});
