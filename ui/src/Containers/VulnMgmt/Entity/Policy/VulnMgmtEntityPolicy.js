import React from 'react';
import gql from 'graphql-tag';

import { workflowEntityPropTypes, workflowEntityDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import entityTypes from 'constants/entityTypes';
import { DEPLOYMENT_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import WorkflowEntityPage from 'Containers/Workflow/WorkflowEntityPage';
import queryService from 'modules/queryService';
import VulnMgmtPolicyOverview from './VulnMgmtPolicyOverview';
import VulnMgmtList from '../../List/VulnMgmtList';

const VulmMgmtEntityPolicy = ({ entityId, entityListType, search, entityContext, sort, page }) => {
    const overviewQuery = gql`
        query getPolicy($id: ID!) {
            result: policy(id: $id) {
                id
                name
                description
                rationale
                remediation
                severity
                policyStatus
                categories
                #lastViolated
                #lastUpdated
                enforcementActions
                lifecycleStages
                fields {
                    addCapabilities
                    args
                    command
                    component {
                        name
                        version
                    }
                    containerResourcePolicy {
                        cpuResourceLimit {
                            op
                            value
                        }
                        cpuResourceRequest {
                            op
                            value
                        }
                        memoryResourceLimit {
                            op
                            value
                        }
                        memoryResourceRequest {
                            op
                            value
                        }
                    }
                    cve
                    cvss {
                        op
                        value
                    }
                    directory
                    disallowedAnnotation {
                        envVarSource
                        key
                        value
                    }
                    dropCapabilities
                    env {
                        envVarSource
                        key
                        value
                    }
                    fixedBy
                    #hostMountPolicy {
                    # no fields defined in schema
                    #}
                    imageName {
                        registry
                        remote
                        tag
                    }
                    lineRule {
                        instruction
                        value
                    }
                    permissionPolicy {
                        permissionLevel
                    }
                    portExposurePolicy {
                        exposureLevels
                    }
                    portPolicy {
                        port
                        protocol
                    }
                    processPolicy {
                        ancestor
                        args
                        name
                        uid
                    }
                    requiredAnnotation {
                        envVarSource
                        key
                        value
                    }
                    requiredLabel {
                        envVarSource
                        key
                        value
                    }
                    #scanAgeDays
                    user
                    volumePolicy {
                        destination
                        name
                        source
                        type
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
                whitelists {
                    deployment {
                        name
                        scope {
                            cluster
                            label {
                                key
                                value
                            }
                            namespace
                        }
                    }
                    expiration
                    image {
                        name
                    }
                    name
                }
                deploymentCount
                deployments {
                    ...deploymentListFields
                }
            }
        }
        ${DEPLOYMENT_LIST_FRAGMENT}
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        return gql`
        query getPolicy${entityListType}($id: ID!, $query: String) {
            result: policy(id: $id) {
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
            query: queryService.objectToWhereClause({
                ...search,
                Category: 'Vulnerability Management'
            })
        }
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
        />
    );
};

VulmMgmtEntityPolicy.propTypes = workflowEntityPropTypes;
VulmMgmtEntityPolicy.defaultProps = workflowEntityDefaultProps;

export default VulmMgmtEntityPolicy;
