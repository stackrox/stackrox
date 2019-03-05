import React from 'react';
import { mount } from 'enzyme';
import { MockedProvider } from 'react-apollo/test-utils';
import { Query } from 'react-apollo';
import getRouterOptions from 'constants/routerOptions';

import { AGGREGATED_RESULTS } from 'queries/controls';
import ComplianceAcrossEntities from './ComplianceAcrossEntities';

const mocks = [
    {
        request: {
            query: AGGREGATED_RESULTS,
            variables: {
                groupBy: ['STANDARD', 'NODE'],
                unit: 'CONTROL'
            }
        }
    },
    {
        request: {
            query: AGGREGATED_RESULTS,
            variables: {
                groupBy: ['STANDARD', 'CLUSTER'],
                unit: 'CONTROL'
            }
        }
    },
    {
        request: {
            query: AGGREGATED_RESULTS,
            variables: {
                groupBy: ['STANDARD', 'NAMESPACE'],
                unit: 'CONTROL'
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
    expect(queryVars.unit === entityType).toBe(true);
};

it('renders for Nodes in Compliance', () => {
    const entityType = 'NODE';
    const element = mount(
        <MockedProvider mocks={mocks} addTypename={false}>
            <ComplianceAcrossEntities entityType={entityType} />
        </MockedProvider>,
        getRouterOptions(jest.fn())
    );

    checkQueryForElement(element, entityType);
});

it('renders for Namespaces in Compliance', () => {
    const entityType = 'NAMESPACE';
    const element = mount(
        <MockedProvider mocks={mocks} addTypename={false}>
            <ComplianceAcrossEntities entityType={entityType} />
        </MockedProvider>,
        getRouterOptions(jest.fn())
    );

    checkQueryForElement(element, entityType);
});

it('renders for Clusters in Compliance', () => {
    const entityType = 'CLUSTER';
    const element = mount(
        <MockedProvider mocks={mocks} addTypename={false}>
            <ComplianceAcrossEntities entityType={entityType} />
        </MockedProvider>,
        getRouterOptions(jest.fn())
    );

    checkQueryForElement(element, entityType);
});
