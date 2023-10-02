import React, { useContext } from 'react';
import { gql } from '@apollo/client';

import useCases from 'constants/useCaseTypes';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import workflowStateContext from 'Containers/workflowStateContext';
import WorkflowEntityPage from '../WorkflowEntityPage';
import {
    vulMgmtPolicyQuery,
    getScopeQuery,
    tryUpdateQueryWithVulMgmtPolicyClause,
} from '../VulnMgmtPolicyQueryUtil';
import VulnMgmtClusterOverview from './VulnMgmtClusterOverview';
import EntityList from '../../List/VulnMgmtList';

const VulmMgmtEntityCluster = ({
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
        query getCluster($id: ID!) {
            result: cluster(id: $id) {
                id
                name
                priority
                #createdAt
                status {
                    orchestratorMetadata {
                        buildDate
                        version
                    }
                }
                istioEnabled
                nodeCount
                namespaceCount
                deploymentCount
                imageCount
                imageComponentCount
                nodeComponentCount
                imageVulnerabilityCount
                nodeVulnerabilityCount
                clusterVulnerabilityCount
            }
        }
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        // @TODO: if we are ever able to search for k8s and istio vulns, swap out this hack for a regular query
        const parsedListFieldName = listFieldName;
        const parsedEntityListType = defaultCountKeyMap[entityListType];
        return gql`
            query getCluster${entityListType}($id: ID!, $pagination: Pagination, $query: String, $policyQuery: String, $scopeQuery: String) {
                result: cluster(id: $id) {
                    id
                    ${parsedEntityListType}(query: $query)
                    ${parsedListFieldName}(query: $query, pagination: $pagination) { ...${fragmentName} }
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
            entityType={entityTypes.CLUSTER}
            entityListType={entityListType}
            useCase={useCases.VULN_MANAGEMENT}
            ListComponent={EntityList}
            OverviewComponent={VulnMgmtClusterOverview}
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

VulmMgmtEntityCluster.propTypes = entityComponentPropTypes;
VulmMgmtEntityCluster.defaultProps = entityComponentDefaultProps;

export default VulmMgmtEntityCluster;
