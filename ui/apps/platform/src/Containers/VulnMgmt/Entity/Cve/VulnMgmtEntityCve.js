import React, { useContext } from 'react';
import { gql } from '@apollo/client';

import { workflowEntityPropTypes, workflowEntityDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import entityTypes, { resourceTypes } from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import workflowStateContext from 'Containers/workflowStateContext';
import {
    VULN_CVE_DETAIL_FRAGMENT,
    IMAGE_CVE_DETAIL_FRAGMENT,
    NODE_CVE_DETAIL_FRAGMENT,
    CLUSTER_CVE_DETAIL_FRAGMENT,
} from 'Containers/VulnMgmt/VulnMgmt.fragments';
import NoResultsMessage from 'Components/NoResultsMessage';
import WorkflowEntityPage from '../WorkflowEntityPage';
import VulnMgmtCveOverview from './VulnMgmtCveOverview';
import VulnMgmtList from '../../List/VulnMgmtList';
import {
    vulMgmtPolicyQuery,
    tryUpdateQueryWithVulMgmtPolicyClause,
    getScopeQuery,
} from '../VulnMgmtPolicyQueryUtil';

const validCVETypes = [
    resourceTypes.CVE,
    resourceTypes.IMAGE_CVE,
    resourceTypes.NODE_CVE,
    resourceTypes.CLUSTER_CVE,
];

// Distinguish GraphQL query name and therefore opname especially for integration tests.
const queryNameMap = {
    CVE: 'getCve',
    IMAGE_CVE: 'getImageCve',
    NODE_CVE: 'getNodeCve',
    CLUSTER_CVE: 'getClusterCve',
};

const vulnQueryMap = {
    CVE: 'vulnerability',
    IMAGE_CVE: 'imageVulnerability',
    NODE_CVE: 'nodeVulnerability',
    CLUSTER_CVE: 'clusterVulnerability',
};
const vulnFieldMap = {
    CVE: VULN_CVE_DETAIL_FRAGMENT,
    IMAGE_CVE: IMAGE_CVE_DETAIL_FRAGMENT,
    NODE_CVE: NODE_CVE_DETAIL_FRAGMENT,
    CLUSTER_CVE: CLUSTER_CVE_DETAIL_FRAGMENT,
};

function getCVETypeFromStack(worklowStateStack) {
    const cveTypes = worklowStateStack.filter((state) => {
        return validCVETypes.includes(state.t);
    });
    if (cveTypes.length) {
        return cveTypes[0].t;
    }
    return undefined;
}

const VulmMgmtCve = ({ entityId, entityListType, search, entityContext, sort, page }) => {
    const workflowState = useContext(workflowStateContext);
    const worklowStateStack = workflowState.getStateStack();
    const cveType = getCVETypeFromStack(worklowStateStack) || entityTypes.IMAGE_CVE;
    const queryName = queryNameMap[cveType];
    const vulnQuery = vulnQueryMap[cveType];
    const vulnFields = vulnFieldMap[cveType];

    // When switching between workflow states, the entity state changes before this component dismounts
    if (!entityListType && (!vulnQuery || !vulnFields)) {
        return (
            <NoResultsMessage
                message={`No vulnerability found of type ${cveType}`}
                className="p-3"
                icon="info"
            />
        );
    }

    const overviewQuery = gql`
        query ${queryName}($id: ID!, $query: String, $scopeQuery: String) {
            result: ${vulnQuery}(id: $id) {
                ...cveFields
            }
        }
        ${vulnFields}
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        return gql`
            query ${queryName}${entityListType}($id: ID!, $pagination: Pagination, $query: String, $policyQuery: String, $scopeQuery: String) {
                result: ${vulnQuery}(id: $id) {
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
            entityType={cveType}
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
