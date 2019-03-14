import React from 'react';
import { mount } from 'enzyme';
import { MockedProvider } from 'react-apollo/test-utils';
import { Query } from 'react-apollo';
import getRouterOptions from 'constants/routerOptions';
import entityTypes, { standardTypes } from 'constants/entityTypes';
import { standardLabels } from 'messages/standards';

import { COMPLIANCE_STANDARDS } from 'queries/standard';
import queryService from 'modules/queryService';
import ComplianceByStandard from './ComplianceByStandard';

const groupBy = [entityTypes.STANDARD, entityTypes.CATEGORY, entityTypes.CONTROL];
const getMock = standardType => {
    const whereClause = {
        Standard: standardLabels[standardType]
    };
    return [
        {
            request: {
                query: COMPLIANCE_STANDARDS,
                variables: {
                    groupBy,
                    where: queryService.objectToWhereClause(whereClause)
                }
            }
        }
    ];
};

const checkQueryForElement = (element, standardType) => {
    const queryProps = element.find(Query).props();
    const queryName = queryProps.query.definitions[0].name.value;
    const queryVars = queryProps.variables;
    expect(queryName === 'complianceStandards').toBe(true);
    expect(queryVars.groupBy).toEqual(groupBy);
    expect(queryVars.where === `Standard:${standardLabels[standardType]}`).toBe(true);
};

const testQueryForEntityType = standardType => {
    const mock = getMock(standardType);
    const element = mount(
        <MockedProvider mocks={mock} addTypename={false}>
            <ComplianceByStandard standardType={standardType} />
        </MockedProvider>,
        getRouterOptions(jest.fn())
    );

    checkQueryForElement(element, standardType);
};

it('renders for Compliance By CIS Docker', () => {
    testQueryForEntityType(standardTypes.CIS_Docker_v1_1_0);
});

it('renders for Compliance By CIS K8s', () => {
    testQueryForEntityType(standardTypes.CIS_Kubernetes_v1_2_0);
});

it('renders for Compliance By HIPAA', () => {
    testQueryForEntityType(standardTypes.HIPAA_164);
});

it('renders for Compliance By NIST', () => {
    testQueryForEntityType(standardTypes.NIST_800_190);
});

it('renders for Compliance By PCI', () => {
    testQueryForEntityType(standardTypes.PCI_DSS_3_2);
});
