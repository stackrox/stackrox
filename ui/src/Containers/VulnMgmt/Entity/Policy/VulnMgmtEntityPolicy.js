import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import gql from 'graphql-tag';

import { workflowEntityPropTypes, workflowEntityDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import entityTypes from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import workflowStateContext from 'Containers/workflowStateContext';
import { DEPLOYMENT_LIST_FRAGMENT } from 'Containers/VulnMgmt/VulnMgmt.fragments';
import WorkflowEntityPage from 'Containers/Workflow/WorkflowEntityPage';
import queryService from 'modules/queryService';
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
    setRefreshTrigger
}) => {
    const queryVarParam = entityContext[entityTypes.POLICY] ? '' : '(query: $scopeQuery)';
    const queryVarConcat = entityContext[entityTypes.POLICY] ? '' : ', query: $scopeQuery';
    const workflowState = useContext(workflowStateContext);

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
                    imageAgeDays
                    scanAgeDays
                    noScanExists
                    readOnlyRootFs
                    whitelistEnabled
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
                deploymentCount${queryVarParam}
                deployments${queryVarParam} {
                    ...deploymentFields
                }
                unusedVarSink(query: $scopeQuery)
            }
        }
        ${DEPLOYMENT_LIST_FRAGMENT}
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        // we don't need to filter the count key or entity list when coming from a specific policy since we're already filtering through policy ID
        // @TODO: rethink entity context and when it accumulates entity info -- currently it holds info from list -> selected row, but not when you
        // hit the external link and view it as an entity page
        return gql`
        query getPolicy${entityListType}($id: ID!, $pagination: Pagination, $query: String, $policyQuery: String, $scopeQuery: String) {
            result: policy(id: $id) {
                id
                ${defaultCountKeyMap[entityListType]}${queryVarParam}
                ${listFieldName}(pagination: $pagination${queryVarConcat}) { ...${fragmentName} }
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
            query: queryService.objectToWhereClause({ ...search }),
            ...vulMgmtPolicyQuery,
            scopeQuery: getScopeQuery(fullEntityContext)
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
            setRefreshTrigger={setRefreshTrigger}
        />
    );
};

VulmMgmtEntityPolicy.propTypes = {
    ...workflowEntityPropTypes,
    setRefreshTrigger: PropTypes.func
};
VulmMgmtEntityPolicy.defaultProps = workflowEntityDefaultProps;

export default VulmMgmtEntityPolicy;
