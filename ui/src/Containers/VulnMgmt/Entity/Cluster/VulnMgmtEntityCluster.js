import React from 'react';
import gql from 'graphql-tag';

import useCases from 'constants/useCaseTypes';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import queryService from 'modules/queryService';
import { VULN_CVE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import WorkflowEntityPage from 'Containers/Workflow/WorkflowEntityPage';
import {
    getPolicyQueryVar,
    tryUpdateQueryWithVulMgmtPolicyClause
} from '../VulnMgmtPolicyQueryUtil';
import VulnMgmtClusterOverview from './VulnMgmtClusterOverview';
import EntityList from '../../List/VulnMgmtList';

const VulmMgmtDeployment = ({ entityId, entityListType, search, sort, page, entityContext }) => {
    const overviewQuery = gql`
        query getCluster($id: ID!, $policyQuery: String) {
            result: cluster(id: $id) {
                id
                name
                priority
                policyStatus(query: $policyQuery) {
                    status
                    failingPolicies {
                        id
                        name
                        description
                        policyStatus
                        latestViolation
                        severity
                        deploymentCount
                        lifecycleStages
                        enforcementActions
                    }
                }
                #createdAt
                status {
                    orchestratorMetadata {
                        version
                    }
                }
                #istioEnabled
                policyCount(query: $policyQuery)
                namespaceCount
                deploymentCount
                imageCount
                componentCount
                vulnCount
                vulnerabilities: vulns {
                    ...cveFields
                }
            }
        }
        ${VULN_CVE_LIST_FRAGMENT}
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        // @TODO: remove this hack after we are able to search for k8s vulns
        const parsedListFieldName =
            search && search['Vulnerability Type'] ? 'vulns: k8sVulns' : listFieldName;
        return gql`
        query getCluster_${entityListType}($id: ID!, $query: String${getPolicyQueryVar(
            entityListType
        )}) {
            result: cluster(id: $id) {
                id
                ${parsedListFieldName}(query: $query) { ...${fragmentName} }
            }
        }
        ${fragment}
    `;
    }

    const queryOptions = {
        variables: {
            id: entityId,
            query: tryUpdateQueryWithVulMgmtPolicyClause(entityListType, search),
            policyQuery: queryService.objectToWhereClause({ Category: 'Vulnerability Management' })
        }
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
        />
    );
};

VulmMgmtDeployment.propTypes = entityComponentPropTypes;
VulmMgmtDeployment.defaultProps = entityComponentDefaultProps;

export default VulmMgmtDeployment;
