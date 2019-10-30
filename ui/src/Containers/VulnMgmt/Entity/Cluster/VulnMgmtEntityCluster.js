import React from 'react';
import gql from 'graphql-tag';

import useCases from 'constants/useCaseTypes';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';
import entityTypes from 'constants/entityTypes';
import queryService from 'modules/queryService';
import { CVE_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import WorkflowEntityPage from 'Containers/Workflow/WorkflowEntityPage';
import VulnMgmtClusterOverview from './VulnMgmtClusterOverview';
import EntityList from '../../List/VulnMgmtList';

const VulmMgmtDeployment = ({ entityId, entityListType, search, sort, page, entityContext }) => {
    const overviewQuery = gql`
        query getCluster($id: ID!) {
            result: cluster(id: $id) {
                id
                name
                priority
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
                #createdAt
                status {
                    orchestratorMetadata {
                        version
                    }
                }
                #istioEnabled
                policyCount
                namespaceCount
                deploymentCount
                imageCount
                imageComponentCount
                vulnCount
                vulnerabilities: vulns {
                    ...cveListFields
                }
            }
        }
        ${CVE_LIST_FRAGMENT}
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        return gql`
        query getCluster_${entityListType}($id: ID!, $query: String) {
            result: cluster(id: $id) {
                id
                ${listFieldName}(query: $query) { ...${fragmentName} }
            }
        }
        ${fragment}
    `;
    }

    const queryOptions = {
        variables: {
            id: entityId,
            query: search ? queryService.objectToWhereClause(search) : null
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
