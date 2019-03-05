import React from 'react';
import { mount } from 'enzyme';
import { MockedProvider } from 'react-apollo/test-utils';
import { Query } from 'react-apollo';
import getRouterOptions from 'constants/routerOptions';

import { CLUSTER_VERSION_QUERY } from 'queries/cluster';
import ClusterVersion from './ClusterVersion';

const clusterId = '1234';

const mocks = [
    {
        request: {
            query: CLUSTER_VERSION_QUERY,
            variables: {
                id: clusterId
            }
        }
    }
];

it('renders without error', () => {
    const element = mount(
        <MockedProvider mocks={mocks} addTypename={false}>
            <ClusterVersion clusterId={clusterId} entityType="CLUSTER" />
        </MockedProvider>,
        getRouterOptions(jest.fn())
    );

    const queryProps = element.find(Query).props();
    const queryName = queryProps.query.definitions[0].name.value;
    const queryVars = queryProps.variables;
    expect(queryName === 'getClusterVersion').toBe(true);
    expect(queryVars.id === clusterId).toBe(true);
});
