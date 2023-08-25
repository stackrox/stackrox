import React, { useContext } from 'react';
import { gql } from '@apollo/client';

import useCases from 'constants/useCaseTypes';
import queryService from 'utils/queryService';
import { workflowEntityPropTypes, workflowEntityDefaultProps } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import workflowStateContext from 'Containers/workflowStateContext';
import WorkflowEntityPage from '../WorkflowEntityPage';
import VulnMgmtNodeOverview from './VulnMgmtNodeOverview';
import EntityList from '../../List/VulnMgmtList';
import {
    vulMgmtPolicyQuery,
    tryUpdateQueryWithVulMgmtPolicyClause,
} from '../VulnMgmtPolicyQueryUtil';

const VulmMgmtNode = ({
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
        query getNode($id: ID!) {
            result: node(id: $id) {
                id
                name
                containerRuntimeVersion
                externalIpAddresses
                internalIpAddresses
                joinedAt
                nodeStatus
                kernelVersion
                kubeletVersion
                osImage
                topVuln: topNodeVulnerability {
                    cvss
                    scoreVersion
                }
                priority
                labels {
                    key
                    value
                }
                annotations {
                    key
                    value
                }
                nodeVulnerabilityCount
                notes
                scan {
                    scanTime
                    notes
                    components {
                        id
                        priority
                        name
                        version
                    }
                }
                ${entityContext[entityTypes.CLUSTER] ? '' : 'clusterId clusterName'}
            }
        }
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        return gql`
            query getNode${entityListType}($id: ID!, $pagination: Pagination, $query: String, $policyQuery: String, $scopeQuery: String) {
                result: node(id: $id) {
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
            scopeQuery: queryService.objectToWhereClause({
                ...queryService.entityContextToQueryObject(fullEntityContext),
                Category: 'Vulnerability Management',
            }),
        },
    };

    return (
        <WorkflowEntityPage
            entityId={entityId}
            entityType={entityTypes.NODE}
            entityListType={entityListType}
            useCase={useCases.VULN_MANAGEMENT}
            ListComponent={EntityList}
            OverviewComponent={VulnMgmtNodeOverview}
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

VulmMgmtNode.propTypes = workflowEntityPropTypes;
VulmMgmtNode.defaultProps = workflowEntityDefaultProps;

export default VulmMgmtNode;
