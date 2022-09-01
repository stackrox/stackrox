import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { gql } from '@apollo/client';

import { workflowEntityPropTypes, workflowEntityDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import entityTypes from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import workflowStateContext from 'Containers/workflowStateContext';
import {
    DEPLOYMENT_LIST_FOR_POLICY_FRAGMENT,
    DEPLOYMENT_LIST_FOR_POLICY_FRAGMENT_UPDATED,
} from 'Containers/VulnMgmt/VulnMgmt.fragments';
import WorkflowEntityPage from 'Containers/Workflow/WorkflowEntityPage';
import useFeatureFlags from 'hooks/useFeatureFlags';
import queryService from 'utils/queryService';
import VulnMgmtPolicyOverview from './VulnMgmtPolicyOverview';
import VulnMgmtList from '../../List/VulnMgmtList';
import { getScopeQuery, vulMgmtPolicyQuery } from '../VulnMgmtPolicyQueryUtil';

const VulmMgmtEntityPolicy = ({
    entityId,
    entityListType,
    search,
    entityContext,
    sort,
    page,
    setRefreshTrigger,
}) => {
    const queryVarParam = entityContext[entityTypes.POLICY] ? '' : '(query: $scopeQuery)';
    const workflowState = useContext(workflowStateContext);

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showVMUpdates = isFeatureFlagEnabled('ROX_FRONTEND_VM_UPDATES');

    const fragmentToUse = showVMUpdates
        ? DEPLOYMENT_LIST_FOR_POLICY_FRAGMENT_UPDATED
        : DEPLOYMENT_LIST_FOR_POLICY_FRAGMENT;

    const overviewQuery = gql`
        query getPolicy($id: ID!, $policyQuery: String, $scopeQuery: String) {
            result: policy(id: $id) {
                id
                name
                description
                disabled
                rationale
                remediation
                severity
                policyStatus${queryVarParam}
                categories
                latestViolation${queryVarParam}
                lastUpdated
                enforcementActions
                lifecycleStages
                isDefault
                policySections {
                    sectionName
                    policyGroups {
                        booleanOperator
                        fieldName
                        negate
                        values {
                            value
                        }
                    }
                }
                scope {
                    cluster
                    label {
                        key
                        value
                    }
                    namespace
                }
                exclusions {
                    deployment {
                        name
                    }
                    image {
                        name
                    }
                }
                deploymentCount${queryVarParam}
                deployments${queryVarParam} {
                    ...deploymentFields
                }
                unusedVarSink(query: $scopeQuery)
            }
        }
        ${fragmentToUse}
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        // we don't need to filter the count key or entity list when coming from a specific policy since we're already filtering through policy ID
        // @TODO: rethink entity context and when it accumulates entity info -- currently it holds info from list -> selected row, but not when you
        // hit the external link and view it as an entity page
        return gql`
        query getPolicy${entityListType}($id: ID!, $pagination: Pagination, $query: String, $policyQuery: String, $scopeQuery: String) {
            result: policy(id: $id) {
                id
                ${defaultCountKeyMap[entityListType]}(query: $query)
                ${listFieldName}(query: $query, pagination: $pagination) { ...${fragmentName} }
                unusedVarSink(query: $policyQuery)
                unusedVarSink(query: $scopeQuery)
                unusedVarSink(query: $query)
            }
        }
        ${fragment}
    `;
    }

    const fullEntityContext = workflowState.getEntityContext();
    const queryOptions = {
        variables: {
            id: entityId,
            query: queryService.objectToWhereClause({
                ...queryService.entityContextToQueryObject(fullEntityContext),
                ...search,
            }),
            ...vulMgmtPolicyQuery,
            scopeQuery: getScopeQuery(fullEntityContext),
        },
    };

    return (
        <WorkflowEntityPage
            entityId={entityId}
            entityType={entityTypes.POLICY}
            entityListType={entityListType}
            useCase={useCases.VULN_MANAGEMENT}
            ListComponent={VulnMgmtList}
            OverviewComponent={VulnMgmtPolicyOverview}
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

VulmMgmtEntityPolicy.propTypes = {
    ...workflowEntityPropTypes,
    setRefreshTrigger: PropTypes.func,
};
VulmMgmtEntityPolicy.defaultProps = workflowEntityDefaultProps;

export default VulmMgmtEntityPolicy;
