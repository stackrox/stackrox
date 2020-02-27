import React from 'react';
import gql from 'graphql-tag';
import WorkflowEntityPage from 'Containers/Workflow/WorkflowEntityPage';
import entityTypes from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import useCases from 'constants/useCaseTypes';
import { VULN_CVE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import EntityList from '../../List/VulnMgmtList';
import VulnMgmtComponentOverview from './VulnMgmtComponentOverview';
import {
    vulMgmtPolicyQuery,
    tryUpdateQueryWithVulMgmtPolicyClause,
    getScopeQuery
} from '../VulnMgmtPolicyQueryUtil';

const VulnMgmtComponent = ({ entityId, entityListType, search, entityContext, sort, page }) => {
    const overviewQuery = gql`
        query getComponent($id: ID!, $query: String, $scopeQuery: String) {
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
                fixableCVEs: vulns(query: "Fixable:true") {
                    ...cveFields
                }
            }
        }
        ${VULN_CVE_LIST_FRAGMENT}
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

    const queryOptions = {
        variables: {
            id: entityId,
            query: tryUpdateQueryWithVulMgmtPolicyClause(entityListType, search, entityContext),
            ...vulMgmtPolicyQuery,
            scopeQuery: getScopeQuery(entityContext)
        }
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
        />
    );
};

export default VulnMgmtComponent;
