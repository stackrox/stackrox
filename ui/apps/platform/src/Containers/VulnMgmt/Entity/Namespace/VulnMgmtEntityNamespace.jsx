import React, { useContext } from 'react';
import { gql } from '@apollo/client';

import useCases from 'constants/useCaseTypes';
import { workflowEntityPropTypes, workflowEntityDefaultProps } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import workflowStateContext from 'Containers/workflowStateContext';
import WorkflowEntityPage from '../WorkflowEntityPage';
import VulnMgmtNamespaceOverview from './VulnMgmtNamespaceOverview';
import EntityList from '../../List/VulnMgmtList';
import {
    vulMgmtPolicyQuery,
    tryUpdateQueryWithVulMgmtPolicyClause,
    getScopeQuery,
} from '../VulnMgmtPolicyQueryUtil';

const VulnMgmtNamespace = ({
    entityId,
    entityListType,
    search,
    sort,
    page,
    entityContext,
    refreshTrigger,
    setRefreshTrigger,
}) => {
    const workflowState = useContext(workflowStateContext);

    const overviewQuery = gql`
        query getNamespace($id: ID!) {
            result: namespace(id: $id) {
                metadata {
                    priority
                    name
                    ${entityContext[entityTypes.CLUSTER] ? '' : 'clusterName clusterId'}
                    id
                    labels {
                        key
                        value
                    }
                }
                deploymentCount
                imageCount
                imageComponentCount
                imageVulnerabilityCount
            }
        }
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        return gql`
        query getNamespace${entityListType}($id: ID!, $pagination: Pagination, $query: String, $policyQuery: String, $scopeQuery: String) {
            result: namespace(id: $id) {
                metadata {
                    id
                }
                ${defaultCountKeyMap[entityListType]}(query: $query)
                ${listFieldName}(query: $query, pagination: $pagination) { ...${fragmentName} }
                unusedVarSink(query: $policyQuery)
                unusedVarSink(query: $scopeQuery)
            }
        }
        ${fragment}
    `;
    }
    const newEntityContext = { ...entityContext, [entityTypes.NAMESPACE]: entityId };

    const fullEntityContext = workflowState.getEntityContext();
    const queryOptions = {
        variables: {
            id: entityId,
            query: tryUpdateQueryWithVulMgmtPolicyClause(entityListType, search, newEntityContext),
            ...vulMgmtPolicyQuery,
            cachebuster: refreshTrigger,
            scopeQuery: getScopeQuery(fullEntityContext),
        },
    };

    return (
        <WorkflowEntityPage
            entityId={entityId}
            entityType={entityTypes.NAMESPACE}
            entityListType={entityListType}
            useCase={useCases.VULN_MANAGEMENT}
            ListComponent={EntityList}
            OverviewComponent={VulnMgmtNamespaceOverview}
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

VulnMgmtNamespace.propTypes = workflowEntityPropTypes;
VulnMgmtNamespace.defaultProps = workflowEntityDefaultProps;

export default VulnMgmtNamespace;
