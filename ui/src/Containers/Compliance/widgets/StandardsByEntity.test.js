import React from 'react';
import { mount } from 'enzyme';
import { MockedProvider } from 'react-apollo/test-utils';
import { Query } from 'react-apollo';
import getRouterOptions from 'constants/routerOptions';
import entityTypes from 'constants/entityTypes';

import { AGGREGATED_RESULTS as QUERY } from 'queries/controls';
import StandardsByEntity from './StandardsByEntity';

const groupBy = [entityTypes.STANDARD, entityTypes.CLUSTER];
const unit = entityTypes.CONTROL;
const mocks = [
    {
        request: {
            query: QUERY,
            variables: {
                groupBy,
                unit
            }
        }
    }
];

const checkQueryForElement = element => {
    const queryProps = element.find(Query).props();
    const queryName = queryProps.query.definitions[0].name.value;
    const queryVars = queryProps.variables;
    expect(queryName === 'getAggregatedResults').toBe(true);
    expect(queryVars.groupBy).toEqual(groupBy);
    expect(queryVars.unit === unit).toBe(true);
};

it('renders for Passing Standards by Cluster', () => {
    const element = mount(
        <MockedProvider mocks={mocks} addTypename={false}>
            <StandardsByEntity entityType={entityTypes.CLUSTER} />
        </MockedProvider>,
        getRouterOptions(jest.fn())
    );

    checkQueryForElement(element);
});
