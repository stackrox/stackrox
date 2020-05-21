import React, { useContext } from 'react';
import gql from 'graphql-tag';

import { workflowEntityPropTypes, workflowEntityDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import entityTypes from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import workflowStateContext from 'Containers/workflowStateContext';
import WorkflowEntityPage from 'Containers/Workflow/WorkflowEntityPage';
import VulnMgmtCveOverview from './VulnMgmtCveOverview';
import VulnMgmtList from '../../List/VulnMgmtList';
import {
    vulMgmtPolicyQuery,
    tryUpdateQueryWithVulMgmtPolicyClause,
    getScopeQuery,
} from '../VulnMgmtPolicyQueryUtil';

const VulmMgmtCve = ({ entityId, entityListType, search, entityContext, sort, page }) => {
    const workflowState = useContext(workflowStateContext);

    const overviewQuery = gql`
        query getCve($id: ID!, $query: String, $scopeQuery: String) {
            result: vulnerability(id: $id) {
                id: cve
                cve
                envImpact
                cvss
                scoreVersion
                link # for View on NVD website
                vectors {
                    __typename
                    ... on CVSSV2 {
                        impactScore
                        exploitabilityScore
                        vector
                    }
                    ... on CVSSV3 {
                        impactScore
                        exploitabilityScore
                        vector
                    }
                }
                publishedOn
                lastModified
                summary
                fixedByVersion
                isFixable(query: $scopeQuery)
                createdAt
                componentCount(query: $query)
                imageCount(query: $query)
                deploymentCount(query: $query)
            }
        }
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        return gql`
        query getCve${entityListType}($id: ID!, $pagination: Pagination, $query: String, $policyQuery: String, $scopeQuery: String) {
            result: vulnerability(id: $id) {
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
            scopeQuery: getScopeQuery(fullEntityContext),
        },
    };

    return (
        <WorkflowEntityPage
            entityId={entityId}
            entityType={entityTypes.CVE}
            entityListType={entityListType}
            useCase={useCases.VULN_MANAGEMENT}
            ListComponent={VulnMgmtList}
            OverviewComponent={VulnMgmtCveOverview}
            overviewQuery={overviewQuery}
            getListQuery={getListQuery}
            search={search}
            sort={sort}
            page={page}
            queryOptions={queryOptions}
            entityContext={entityContext}
        />
    );
};

VulmMgmtCve.propTypes = workflowEntityPropTypes;
VulmMgmtCve.defaultProps = workflowEntityDefaultProps;

export default VulmMgmtCve;
