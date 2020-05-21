import React, { useContext } from 'react';
import gql from 'graphql-tag';

import useCases from 'constants/useCaseTypes';
import entityTypes from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import workflowStateContext from 'Containers/workflowStateContext';
import WorkflowEntityPage from 'Containers/Workflow/WorkflowEntityPage';
import EntityList from '../../List/VulnMgmtList';
import VulnMgmtComponentOverview from './VulnMgmtComponentOverview';
import {
    vulMgmtPolicyQuery,
    tryUpdateQueryWithVulMgmtPolicyClause,
    getScopeQuery,
} from '../VulnMgmtPolicyQueryUtil';

const VulnMgmtComponent = ({
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
        query getComponent($id: ID!, $query: String) {
            result: component(id: $id) {
                id
                name
                version
                priority
                vulnCount(query: $query)
                deploymentCount(query: $query)
                imageCount(query: $query)
                topVuln {
                    cvss
                    scoreVersion
                }
            }
        }
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        return gql`
        query getComponentSubEntity${entityListType}($id: ID!, $pagination: Pagination, $query: String, $policyQuery: String, $scopeQuery: String) {
            result: component(id: $id) {
                id
                ${defaultCountKeyMap[entityListType]}(query: $query)
                ${listFieldName}(query: $query, pagination: $pagination) { ...${fragmentName} }
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
            entityType={entityTypes.COMPONENT}
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

export default VulnMgmtComponent;
