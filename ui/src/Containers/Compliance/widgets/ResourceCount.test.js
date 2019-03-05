import React from 'react';
import { mount } from 'enzyme';
import { MockedProvider } from 'react-apollo/test-utils';
import { Query } from 'react-apollo';
import getRouterOptions from 'constants/routerOptions';

import { NODES_BY_CLUSTER } from 'queries/node';
import ResourceCount from './ResourceCount';

const id = '1234';
const mocks = [
    {
        request: {
            query: NODES_BY_CLUSTER,
            variables: {
                id
            }
        }
    }
];

it('renders without error', () => {
    const element = mount(
        <MockedProvider mocks={mocks} addTypename={false}>
            <ResourceCount
                resourceType="NODE"
                relatedToResourceType="CLUSTER"
                relatedToResourceId={id}
            />
        </MockedProvider>,
        getRouterOptions(jest.fn())
    );

    const queryProps = element.find(Query).props();
    const queryName = queryProps.query.definitions[0].name.value;
    const queryVars = queryProps.variables;
    expect(queryName === 'getNodesByCluster').toBe(true);
    expect(queryVars.id === id).toBe(true);
});
