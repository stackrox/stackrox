import React from 'react';
import { mount } from 'enzyme';
import { MockedProvider } from 'react-apollo/test-utils';
import { Query } from 'react-apollo';
import ReactRouterEnzymeContext from 'react-router-enzyme-context';
import entityTypes from 'constants/entityTypes';

import { AGGREGATED_RESULTS } from 'queries/controls';
import queryService from 'modules/queryService';
import EntityCompliance from './EntityCompliance';

const unit = entityTypes.CONTROL;
const getMock = (entityType, entityName, clusterName) => {
    const whereClause = {
        [entityType]: entityName,
        [entityTypes.CLUSTER]: clusterName
    };
    return [
        {
            request: {
                query: AGGREGATED_RESULTS,
                variables: {
                    groupBy: [entityTypes.STANDARD, entityType],
                    unit,
                    where: queryService.objectToWhereClause(whereClause)
                }
            }
        }
    ];
};

const checkQueryForElement = (mock, element, entityType) => {
    const queryProps = element.find(Query).props();
    const queryName = queryProps.query.definitions[0].name.value;
    const queryVars = queryProps.variables;
    expect(queryName === 'getAggregatedResults').toBe(true);
    expect(queryVars.groupBy).toEqual(['STANDARD', entityType]);
    expect(queryVars.unit === unit).toBe(true);
    expect(queryVars.where === mock.request.variables.where).toBe(true);
};

const testQueryForEntityType = (entityType, entityName, clusterName, id) => {
    const options = new ReactRouterEnzymeContext();
    const mock = getMock(entityType, entityName, clusterName);
    const element = mount(
        <MockedProvider mocks={mock} addTypename={false}>
            <EntityCompliance
                entityType={entityType}
                entityId={id}
                entityName={entityName}
                clusterName={clusterName}
            />
        </MockedProvider>,
        options.get()
    );

    checkQueryForElement(mock[0], element, entityType);
};

it('renders for Cluster Compliance', () => {
    const entityType = entityTypes.CLUSTER;
    const cluster = {
        id: '1234',
        name: 'remote'
    };
    const { name: entityName, id } = cluster;
    const clusterName = entityName;
    testQueryForEntityType(entityType, entityName, clusterName, id);
});

it('renders for Namespace Compliance', () => {
    const entityType = entityTypes.NAMESPACE;
    const namespace = {
        id: '1234a',
        name: 'namespace1',
        clusterName: 'remote'
    };
    const { name: entityName, id } = namespace;
    const { clusterName } = namespace;
    testQueryForEntityType(entityType, entityName, clusterName, id);
});

it('renders for Node Compliance', () => {
    const entityType = entityTypes.NODE;
    const node = {
        id: '1234b',
        name: 'node1',
        clusterName: 'remote'
    };
    const { clusterName, name: entityName, id } = node;
    testQueryForEntityType(entityType, entityName, clusterName, id);
});
