import React from 'react';
import { MockedProvider } from '@apollo/client/testing';
import { screen } from '@testing-library/react';
import '@testing-library/jest-dom/extend-expect';

import workflowStateContext from 'Containers/workflowStateContext';
import parseURL from 'utils/URLParser';
import renderWithRouter from 'test-utils/renderWithRouter';
import RecentlyDetectedVulnerabilities, {
    RECENTLY_DETECTED_VULNERABILITIES,
} from './RecentlyDetectedImageVulnerabilities';

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

        renderWithRouter(
            <MockedProvider mocks={mocks} addTypename={false}>
                <workflowStateContext.Provider value={workflowState}>
                    <RecentlyDetectedVulnerabilities />
                </workflowStateContext.Provider>
            </MockedProvider>
        );

        const messageElement = await screen.findByTestId(messageTestId);
        expect(messageElement.textContent).toContain(expectedMessage);
    });
});
