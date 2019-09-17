import React from 'react';
import { MockedProvider } from '@apollo/react-testing';
import { waitForElement } from '@testing-library/react';
import { CLUSTER_VERSION_QUERY } from 'queries/cluster';
import renderWithRouter from 'test-utils/renderWithRouter';
import '@testing-library/jest-dom/extend-expect';
import ClusterVersion from './ClusterVersion';

const clusterId = '1234';
const mocks = [
    {
        request: {
            query: CLUSTER_VERSION_QUERY,
            variables: {
                id: clusterId
            }
        },
        result: {
            data: {
                cluster: {
                    id: '180ed8f0-193d-4f42-83ec-0e12d707d2f6',
                    name: 'remote',
                    type: 'KUBERNETES_CLUSTER',
                    status: {
                        orchestratorMetadata: {
                            version: 'v1.12.8-gke.10',
                            buildDate: '2019-06-19T20:48:40Z',
                            __typename: 'OrchestratorMetadata'
                        },
                        __typename: 'ClusterStatus'
                    },
                    __typename: 'Cluster'
                }
            }
        }
    }
];

it('renders without error', async () => {
    const { getByTestId } = renderWithRouter(
        <MockedProvider mocks={mocks} addTypename={false}>
            <ClusterVersion clusterId={clusterId} entityType="CLUSTER" />
        </MockedProvider>,
        { route: '/some-route' }
    );
    await waitForElement(() => getByTestId('cluster-version'));
    expect(getByTestId('cluster-version')).toHaveTextContent('v1.12.8-gke.10');
});
