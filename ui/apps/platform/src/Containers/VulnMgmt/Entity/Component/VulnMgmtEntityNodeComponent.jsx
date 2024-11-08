import React, { useContext } from 'react';
import { gql } from '@apollo/client';

import useCases from 'constants/useCaseTypes';
import entityTypes from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import workflowStateContext from 'Containers/workflowStateContext';
import WorkflowEntityPage from '../WorkflowEntityPage';
import EntityList from '../../List/VulnMgmtList';
import VulnMgmtComponentOverview from './VulnMgmtComponentOverview';
import {
    vulMgmtPolicyQuery,
    tryUpdateQueryWithVulMgmtPolicyClause,
    getScopeQuery,
} from '../VulnMgmtPolicyQueryUtil';

// We want to override some values because the nodeComponent object has different field names
export const nodeComponentCountKeyMap = {
    ...defaultCountKeyMap,
    [entityTypes.CVE]: 'vulnCount: nodeVulnerabilityCount',
    [entityTypes.NODE_CVE]: 'nodeVulnerabilityCount',
};

const VulnMgmtEntityNodeComponent = ({
    entityId,
    entityListType,
    search,
    entityContext,
    sort,
    page,
    refreshTrigger,
    setRefreshTrigger,
}) => {
    const workflowState = useContext(workflowStateContext);

    const overviewQuery = gql`
        query getNodeComponent($id: ID!, $query: String, $scopeQuery: String) {
            result: nodeComponent(id: $id) {
                id
                name
                version
                fixedIn
                location(query: $scopeQuery)
                priority
                nodeVulnerabilityCount(query: $query, scopeQuery: $scopeQuery)
                nodeCount(query: $query)
                topVuln: topNodeVulnerability {
                    cvss
                    scoreVersion
                }
                operatingSystem
            }
        }
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        return gql`
            query getNodeComponent${entityListType}($id: ID!, $pagination: Pagination, $query: String, $policyQuery: String, $scopeQuery: String) {
                result: nodeComponent(id: $id) {
                    id
                    ${nodeComponentCountKeyMap[entityListType]}(query: $query, scopeQuery: $scopeQuery)
                    ${listFieldName}(query: $query, scopeQuery: $scopeQuery, pagination: $pagination) { ...${fragmentName} }
                    unusedVarSink(query: $policyQuery)
                    unusedVarSink(query: $scopeQuery)
                }
            }
            ${fragment}
        `;
    }

    const fullEntityContext = workflowState.getEntityContext();
    const queryOptions = {
        variables: {
            id: entityId,
            query: tryUpdateQueryWithVulMgmtPolicyClause(entityListType, search, entityContext),
            ...vulMgmtPolicyQuery,
            cachebuster: refreshTrigger,
            scopeQuery: getScopeQuery(fullEntityContext),
        },
    };

    return (
        <WorkflowEntityPage
            entityId={entityId}
            entityType={entityTypes.NODE_COMPONENT}
            entityListType={entityListType}
            useCase={useCases.VULN_MANAGEMENT}
            ListComponent={EntityList}
            OverviewComponent={VulnMgmtComponentOverview}
            overviewQuery={overviewQuery}
            getListQuery={getListQuery}
            search={search}
            sort={sort}
            page={page}
            queryOptions={queryOptions}
            entityContext={entityContext}
            setRefreshTrigger={setRefreshTrigger}
        />
    );
};

export default VulnMgmtEntityNodeComponent;
