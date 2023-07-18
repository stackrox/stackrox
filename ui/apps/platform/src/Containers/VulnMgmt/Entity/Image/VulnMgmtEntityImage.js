import React, { useContext } from 'react';
import { gql } from '@apollo/client';

import { workflowEntityPropTypes, workflowEntityDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import entityTypes from 'constants/entityTypes';
import { defaultCountKeyMap } from 'constants/workflowPages.constants';
import workflowStateContext from 'Containers/workflowStateContext';
import WorkflowEntityPage from '../WorkflowEntityPage';
import {
    VULN_CVE_ONLY_FRAGMENT,
    VULN_IMAGE_COMPONENT_ACTIVE_STATUS_LIST_FRAGMENT,
} from '../../VulnMgmt.fragments';
import VulnMgmtImageOverview from './VulnMgmtImageOverview';
import EntityList from '../../List/VulnMgmtList';
import {
    vulMgmtPolicyQuery,
    tryUpdateQueryWithVulMgmtPolicyClause,
    getScopeQuery,
} from '../VulnMgmtPolicyQueryUtil';

const VulnMgmtImage = ({
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
        query getImage($id: ID!, $query: String, $scopeQuery: String) {
            result: image(id: $id) {
                id
                lastUpdated
                ${entityContext[entityTypes.DEPLOYMENT] ? '' : 'deploymentCount(query: $query)'}
                metadata {
                    layerShas
                    v1 {
                        created
                        layers {
                            instruction
                            created
                            value
                        }
                    }
                }
                notes
                imageVulnerabilityCount(query: $query)
                priority
                topVuln: topImageVulnerability {
                    cvss
                    scoreVersion
                }
                name {
                    fullName
                    registry
                    remote
                    tag
                }
                scan {
                    scanTime
                    operatingSystem
                    dataSource {
                        name
                    }
                    notes
                    components: imageComponents {
                        id
                        priority
                        name
                        layerIndex
                        version
                        source
                        location
                        vulns: imageVulnerabilities {
                            ...cveFields
                        }
                    }
                }
            }
        }
        ${VULN_CVE_ONLY_FRAGMENT}
    `;

    function getListQuery(listFieldName, fragmentName, fragment) {
        const activeStatusFragment = VULN_IMAGE_COMPONENT_ACTIVE_STATUS_LIST_FRAGMENT;
        const fragmentToUse =
            fragmentName === 'componentFields' || fragmentName === 'imageComponentFields'
                ? activeStatusFragment
                : fragment;
        return gql`
        query getImage${entityListType}($id: ID!, $pagination: Pagination, $query: String, $policyQuery: String, $scopeQuery: String) {
            result: image(id: $id) {
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
            query: tryUpdateQueryWithVulMgmtPolicyClause(entityListType, search),
            ...vulMgmtPolicyQuery,
            cachebuster: refreshTrigger,
            scopeQuery: getScopeQuery(fullEntityContext),
        },
    };

    return (
        <WorkflowEntityPage
            entityId={entityId}
            entityType={entityTypes.IMAGE}
            entityListType={entityListType}
            useCase={useCases.VULN_MANAGEMENT}
            ListComponent={EntityList}
            OverviewComponent={VulnMgmtImageOverview}
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

VulnMgmtImage.propTypes = workflowEntityPropTypes;
VulnMgmtImage.defaultProps = workflowEntityDefaultProps;

export default VulnMgmtImage;
