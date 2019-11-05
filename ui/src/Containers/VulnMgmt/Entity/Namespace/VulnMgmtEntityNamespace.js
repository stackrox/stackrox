import React from 'react';
import gql from 'graphql-tag';

import useCases from 'constants/useCaseTypes';
import { workflowEntityPropTypes, workflowEntityDefaultProps } from 'constants/entityPageProps';
import queryService from 'modules/queryService';
import entityTypes from 'constants/entityTypes';
import WorkflowEntityPage from 'Containers/Workflow/WorkflowEntityPage';
import { VULN_CVE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import VulnMgmtNamespaceOverview from './VulnMgmtNamespaceOverview';
import EntityList from '../../List/VulnMgmtList';

const VulnMgmtNamespace = ({ entityId, entityListType, search, entityContext, sort, page }) => {
    const overviewQuery = gql`
        query getNamespace($id: ID!) {
            result: namespace(id: $id) {
                metadata {
                    priority
                    name
                    clusterName
                    id
                    labels {
                        key
                        value
                    }
                }
                policyStatus {
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
                policyCount
                vulnCount
                deploymentCount
                imageCount
                imageComponentCount
                vulnerabilities: vulns {
                    ...cveFields
                }
            }
        }
        ${VULN_CVE_LIST_FRAGMENT}
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        return gql`
        query getNamespace${entityListType}($id: ID!, $query: String) {
            result: namespace(id: $id) {
                metadata {
                    id
                }
                ${listFieldName}(query: $query) { ...${fragmentName} }
            }
        }
        ${fragment}
    `;
    }

    const queryOptions = {
        variables: {
            id: entityId,
            query: queryService.objectToWhereClause(search)
        }
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
        />
    );
};

VulnMgmtNamespace.propTypes = workflowEntityPropTypes;
VulnMgmtNamespace.defaultProps = workflowEntityDefaultProps;

export default VulnMgmtNamespace;
