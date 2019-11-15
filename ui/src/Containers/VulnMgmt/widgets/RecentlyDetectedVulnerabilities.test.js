import React from 'react';
import { MockedProvider } from '@apollo/react-testing';
import { waitForElement } from '@testing-library/react';
import '@testing-library/jest-dom/extend-expect';

import workflowStateContext from 'Containers/workflowStateContext';
import parseURL from 'modules/URLParser';
import renderWithRouter from 'test-utils/renderWithRouter';
import RecentlyDetectedVulnerabilities, {
    RECENTLY_DETECTED_VULNERABILITIES
} from './RecentlyDetectedVulnerabilities';

const mocks = [
    {
        request: {
            query: RECENTLY_DETECTED_VULNERABILITIES,
            variables: {
                query: ''
            }
        },
        result: {
            data: {
                results: []
            }
        }
    }
];

// ensure you're resetting modules before each test
beforeEach(() => {
    jest.resetModules();
});

describe('RecentlyDetectedVulnerabilities', () => {
    it('should render No vulnerabilities found when the query returns an empty result set', async () => {
        const location = {
            search: '',
            pathname: '/main/vulnerability-management'
        };
        const dateTestId = 'results-message';
        const expectedMessage = 'No vulnerabilities found';

        const workflowState = parseURL(location);

        const { getByTestId } = renderWithRouter(
            <MockedProvider mocks={mocks} addTypename={false}>
                <workflowStateContext.Provider value={workflowState}>
                    <RecentlyDetectedVulnerabilities />
                </workflowStateContext.Provider>
            </MockedProvider>
        );

        await waitForElement(() => getByTestId(dateTestId));
        expect(getByTestId(dateTestId).textContent).toContain(expectedMessage);
    });
});
