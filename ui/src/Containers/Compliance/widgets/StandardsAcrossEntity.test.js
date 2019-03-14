import React from 'react';
import { mount } from 'enzyme';
import { MockedProvider } from 'react-apollo/test-utils';
import { Query } from 'react-apollo';
import getRouterOptions from 'constants/routerOptions';
import entityTypes from 'constants/entityTypes';

import { AGGREGATED_RESULTS } from 'queries/controls';
import StandardsAcrossEntity from './StandardsAcrossEntity';

const unit = entityTypes.CONTROL;
const getMock = entityType => [
    {
        request: {
            query: AGGREGATED_RESULTS,
            variables: {
                groupBy: [entityTypes.STANDARD, entityType],
                unit
            }
        }
    }
];

const checkQueryForElement = (element, entityType) => {
    const queryProps = element.find(Query).props();
    const queryName = queryProps.query.definitions[0].name.value;
    const queryVars = queryProps.variables;
    expect(queryName === 'getAggregatedResults').toBe(true);
    expect(queryVars.groupBy).toEqual(['STANDARD', entityType]);
    expect(queryVars.unit === unit).toBe(true);
};

const testQueryForEntityType = entityType => {
    const mock = getMock(entityType);
    const element = mount(
        <MockedProvider mocks={mock} addTypename={false}>
            <StandardsAcrossEntity entityType={entityType} />
        </MockedProvider>,
        getRouterOptions(jest.fn())
    );

    checkQueryForElement(element, entityType);
};

it('renders for Passing Standards Across Clusters', () => {
    testQueryForEntityType(entityTypes.CLUSTER);
});

it('renders for Passing Standards Across Namespaces', () => {
    testQueryForEntityType(entityTypes.NAMESPACE);
});

it('renders for Passing Standards Across Nodes', () => {
    testQueryForEntityType(entityTypes.NODE);
});
