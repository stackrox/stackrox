import React from 'react';
import { MockedProvider } from '@apollo/client/testing';
import { screen } from '@testing-library/react';

import { CLUSTER_VERSION_QUERY } from 'queries/cluster';
import renderWithRouter from 'test-utils/renderWithRouter';
import ClusterVersion from './ClusterVersion';

const clusterId = '1234';
const k8sMocks = [
    {
        request: {
            query: CLUSTER_VERSION_QUERY,
            variables: {
                id: clusterId,
            },
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
                            openshiftVersion: '',
                            buildDate: '2019-06-19T20:48:40Z',
                            __typename: 'OrchestratorMetadata',
                        },
                        __typename: 'ClusterStatus',
                    },
                    __typename: 'Cluster',
                },
            },
        },
    },
];
const openshiftMocks = [
    {
        request: {
            query: CLUSTER_VERSION_QUERY,
            variables: {
                id: clusterId,
            },
        },
        result: {
            data: {
                cluster: {
                    id: 'a365a914-74c7-46b4-b2c0-77d86f2d6e64',
                    name: 'remote',
                    type: 'OPENSHIFT4_CLUSTER',
                    status: {
                        orchestratorMetadata: {
                            version: 'v1.20.0+bbbc079',
                            openshiftVersion: '4.7.32',
                            buildDate: '2021-09-17T20:21:04Z',
                            __typename: 'OrchestratorMetadata',
                        },
                        __typename: 'ClusterStatus',
                    },
                    __typename: 'Cluster',
                },
            },
        },
    },
];
const openshiftVersionMissingMocks = [
    {
        request: {
            query: CLUSTER_VERSION_QUERY,
            variables: {
                id: clusterId,
            },
        },
        result: {
            data: {
                cluster: {
                    id: 'a365a914-74c7-46b4-b2c0-77d86f2d6e64',
                    name: 'remote',
                    type: 'OPENSHIFT4_CLUSTER',
                    status: {
                        orchestratorMetadata: {
                            version: 'v1.20.0+bbbc079',
                            openshiftVersion: '',
                            buildDate: '2021-09-17T20:21:04Z',
                            __typename: 'OrchestratorMetadata',
                        },
                        __typename: 'ClusterStatus',
                    },
                    __typename: 'Cluster',
                },
            },
        },
    },
];

describe('Compliance ClusterVersion widget', () => {
    it('renders Kubernetes cluster without error', async () => {
        renderWithRouter(
            <MockedProvider mocks={k8sMocks} addTypename={false}>
                <ClusterVersion clusterId={clusterId} entityType="CLUSTER" />
            </MockedProvider>,
            { route: '/some-route' }
        );
        const clusterVersionElement = await screen.findByTestId('cluster-version');
        expect(clusterVersionElement).toHaveTextContent('v1.12.8-gke.10');
    });
    it('renders Openshift cluster without error', async () => {
        renderWithRouter(
            <MockedProvider mocks={openshiftMocks} addTypename={false}>
                <ClusterVersion clusterId={clusterId} entityType="CLUSTER" />
            </MockedProvider>,
            { route: '/some-route' }
        );
        const clusterVersionElement = await screen.findByTestId('cluster-version');
        expect(clusterVersionElement).toHaveTextContent('4.7.32');
    });
    it('renders Openshift with missing version cluster without error', async () => {
        renderWithRouter(
            <MockedProvider mocks={openshiftVersionMissingMocks} addTypename={false}>
                <ClusterVersion clusterId={clusterId} entityType="CLUSTER" />
            </MockedProvider>,
            { route: '/some-route' }
        );
        const clusterVersionElement = await screen.findByTestId('cluster-version');
        expect(clusterVersionElement).toHaveTextContent('OpenShift version cannot be determined');
    });
});
