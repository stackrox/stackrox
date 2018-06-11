import { call } from 'redux-saga/effects';
import { expectSaga } from 'redux-saga-test-plan';
import { dynamic } from 'redux-saga-test-plan/providers';

import { fetchClusters } from 'services/ClustersService';
import { actions } from 'reducers/clusters';
import saga from './clusterSagas';
import createLocationChange from './sagaTestUtils';

describe('Cluster Sagas Test', () => {
    it('should fetch clusters when location changes to dashboard, integrations, policies, compliance', () => {
        const clusters = ['cluster1', 'cluster2'];
        const fetchMock = jest.fn().mockReturnValue({ response: clusters });
        return expectSaga(saga)
            .provide([[call(fetchClusters), dynamic(fetchMock)]])
            .put(actions.fetchClusters.success(clusters))
            .dispatch(createLocationChange('/main/dashboard'))
            .dispatch(createLocationChange('/main/integrations'))
            .dispatch(createLocationChange('/main/policies'))
            .dispatch(createLocationChange('/main/compliance'))
            .silentRun()
            .then(() => {
                expect(fetchMock.mock.calls.length).toBe(4);
            });
    });

    it("shouldn't do a service call to get clusters when location changes to violations, images, risk", () => {
        const fetchMock = jest.fn();
        return expectSaga(saga)
            .provide([[call(fetchClusters), dynamic(fetchMock)]])
            .dispatch(createLocationChange('/main/violations'))
            .dispatch(createLocationChange('/main/images'))
            .dispatch(createLocationChange('/main/risk'))
            .silentRun()
            .then(() => {
                expect(fetchMock.mock.calls.length).toBe(0);
            });
    });
});
