import React from 'react';
import { mount } from 'enzyme';
import { MockedProvider } from 'react-apollo/test-utils';
import { Query } from 'react-apollo';
import getRouterOptions from 'constants/routerOptions';

import ResourceCount from './ResourceCount';

const id = '1234';
const name = 'myNodeName';

const mocks = [
    {
        request: {
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
                relatedToResource={{ id, name }}
            />
        </MockedProvider>,
        getRouterOptions(jest.fn())
    );

    const queryProps = element.find(Query).props();
    const queryVars = queryProps.variables;
    expect(queryVars.query === `Cluster ID:${id}`).toBe(true);
});
