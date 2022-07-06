import React from 'react';
import { MockedProvider } from '@apollo/client/testing';
import { screen } from '@testing-library/react';
import '@testing-library/jest-dom/extend-expect';
import { Provider } from 'react-redux';
import { createBrowserHistory as createHistory } from 'history';
import configureStore from 'store/configureStore';

import workflowStateContext from 'Containers/workflowStateContext';
import parseURL from 'utils/URLParser';
import renderWithRouter from 'test-utils/renderWithRouter';
import RecentlyDetectedVulnerabilities, {
    RECENTLY_DETECTED_VULNERABILITIES,
} from './RecentlyDetectedImageVulnerabilities';

const history = createHistory();
const initialStore = {
    app: {
        featureFlags: {
            featureFlags: [
                {
                    name: 'Enable Frontend VM Updates',
                    envVar: 'ROX_FRONTEND_VM_UPDATES',
                    enabled: false,
                },
            ],
        },
    },
};

const mocks = [
    {
        request: {
            query: RECENTLY_DETECTED_VULNERABILITIES,
            variables: {
                query: 'CVE Type:IMAGE_CVE',
                scopeQuery: '',
                pagination: {
                    offset: 0,
                    limit: 5,
                    sortOption: { field: 'CVE Created Time', reversed: true },
                },
            },
        },
        result: {
            data: {
                results: [],
            },
        },
    },
];

// ensure you're resetting modules before each test
beforeEach(() => {
    jest.resetModules();
});

describe('RecentlyDetectedVulnerabilities', () => {
    it('should render No vulnerabilities found when the query returns an empty result set', async () => {
        const location = {
            search: '',
            pathname: '/main/vulnerability-management',
        };
        const messageTestId = 'results-message';
        const expectedMessage = 'No vulnerabilities found';

        const workflowState = parseURL(location);

        const store = configureStore(initialStore, history);

        renderWithRouter(
            <Provider store={store}>
                <MockedProvider mocks={mocks} addTypename={false}>
                    <workflowStateContext.Provider value={workflowState}>
                        <RecentlyDetectedVulnerabilities />
                    </workflowStateContext.Provider>
                </MockedProvider>
            </Provider>
        );

        const messageElement = await screen.findByTestId(messageTestId);
        expect(messageElement.textContent).toContain(expectedMessage);
    });
});
