import React, { useContext } from 'react';
import { gql } from '@apollo/client';

import useCases from 'constants/useCaseTypes';
import queryService from 'utils/queryService';
import { workflowEntityPropTypes, workflowEntityDefaultProps } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import { VULN_IMAGE_COMPONENT_ACTIVE_STATUS_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import workflowStateContext from 'Containers/workflowStateContext';
import WorkflowEntityPage from '../WorkflowEntityPage';
import VulnMgmtDeploymentOverview from './VulnMgmtDeploymentOverview';
import EntityList from '../../List/VulnMgmtList';
import {
    vulMgmtPolicyQuery,
    tryUpdateQueryWithVulMgmtPolicyClause,
} from '../VulnMgmtPolicyQueryUtil';

const VulmMgmtDeployment = ({
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
        query getDeployment($id: ID!, $policyQuery: String, $scopeQuery: String) {
            result: deployment(id: $id) {
                id
                priority
                policyStatus(query: $scopeQuery)
                failingPolicies(query: $scopeQuery) {
                    id
                    name
                    description
                    policyStatus
                    latestViolation
                    severity
                    lifecycleStages
                    enforcementActions
                    notifiers
                    lastUpdated
                }
                annotations {
                    key
                    value
                }
                ${entityContext[entityTypes.CLUSTER] ? '' : 'cluster { id name }'}
                inactive
                labels {
                    key
                    value
                }
                name
                ${entityContext[entityTypes.NAMESPACE] ? '' : 'namespace namespaceId'}
                priority
                failingPolicyCount(query: $scopeQuery)
                policyCount(query: $policyQuery)
                type
                created
                imageCount
                imageComponentCount
                imageVulnerabilityCount
            }
        }
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        const activeStatusFragment = VULN_IMAGE_COMPONENT_ACTIVE_STATUS_LIST_FRAGMENT;
        const fragmentToUse =
            fragmentName === 'componentFields' || fragmentName === 'imageComponentFields'
                ? activeStatusFragment
                : fragment;
        return gql`
        query getDeployment${entityListType}($id: ID!, $pagination: Pagination, $query: String, $policyQuery: String, $scopeQuery: String) {
            result: deployment(id: $id) {
                id
                ${defaultCountKeyMap[entityListType]}(query: $query)
                ${listFieldName}(query: $query, pagination: $pagination) { ...${fragmentName} }
                unusedVarSink(query: $policyQuery)
                unusedVarSink(query: $scopeQuery)
            }
        }
        ${fragmentToUse}
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
        fetchPolicy: 'no-cache',
    };

    return (
        <WorkflowEntityPage
            entityId={entityId}
            entityType={entityTypes.DEPLOYMENT}
            entityListType={entityListType}
            useCase={useCases.VULN_MANAGEMENT}
            ListComponent={EntityList}
            OverviewComponent={VulnMgmtDeploymentOverview}
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

VulmMgmtDeployment.propTypes = workflowEntityPropTypes;
VulmMgmtDeployment.defaultProps = workflowEntityDefaultProps;

export default VulmMgmtDeployment;
